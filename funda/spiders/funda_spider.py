import re
import scrapy
from scrapy.spiders import CrawlSpider, Rule
from scrapy.linkextractors import LinkExtractor
from funda.items import FundaItem
from scrapy.http import Request
import json

class FundaSpider(CrawlSpider):
    name = "funda_spider"
    allowed_domains = ["funda.nl"]
    
    # List of user agents to rotate
    user_agents = [
        'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
        'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0',
    ]

    def __init__(self, place='amsterdam'):
        super(FundaSpider, self).__init__()
        self.place = place.lower()
        self.start_urls = [f"https://www.funda.nl/zoeken/koop/?selected_area=%5B%22{place}%22%5D"]
        
        # Define rules for following links
        self.rules = (
            Rule(
                LinkExtractor(
                    allow=r'/koop/[^/]+/(?:huis|appartement)-[^/]+/\d+/',
                    deny=(r'/en/', r'/verkocht/', r'/print/', r'/kenmerken/', r'/fotos/', r'/video/')
                ),
                callback='parse_house',
                follow=True
            ),
            # Add pagination rule
            Rule(
                LinkExtractor(
                    allow=r'/zoeken/koop/.*p\d+',
                    deny=(r'/verkocht/', r'/print/', r'/kenmerken/', r'/fotos/', r'/video/')
                ),
                follow=True
            ),
        )
        
        # This is crucial for CrawlSpider
        super()._compile_rules()

    def start_requests(self):
        headers = {
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
            'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
            'Accept-Language': 'nl,en-US;q=0.7,en;q=0.3',
            'Accept-Encoding': 'gzip, deflate, br',
            'DNT': '1',
            'Connection': 'keep-alive',
            'Upgrade-Insecure-Requests': '1',
            'Sec-Fetch-Dest': 'document',
            'Sec-Fetch-Mode': 'navigate',
            'Sec-Fetch-Site': 'none',
            'Sec-Fetch-User': '?1',
            'Cache-Control': 'max-age=0',
            'Referer': 'https://www.funda.nl/',
            'sec-ch-ua': '"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"',
            'sec-ch-ua-mobile': '?0',
            'sec-ch-ua-platform': '"macOS"'
        }
        for url in self.start_urls:
            yield Request(
                url=url,
                headers=headers,
                dont_filter=True,
                meta={
                    'dont_redirect': True,
                    'handle_httpstatus_list': [302, 403],
                    'download_timeout': 30
                },
                errback=self.errback_httpbin,
                callback=self.initial_parse
            )

    def errback_httpbin(self, failure):
        self.logger.error(f"Request failed: {failure.value}")

    def check_if_blocked(self, response):
        # Check response status
        if response.status == 403:
            self.logger.error(f"Blocked (403) on URL: {response.url}")
            return True
        
        # Check for CAPTCHA in the response
        if "captcha" in response.text.lower():
            self.logger.error(f"CAPTCHA detected on URL: {response.url}")
            return True
        
        # Check for specific blocking messages
        blocking_phrases = [
            "access denied",
            "blocked",
            "too many requests",
            "rate limit exceeded"
        ]
        
        for phrase in blocking_phrases:
            if phrase in response.text.lower():
                self.logger.error(f"Blocking phrase '{phrase}' found on URL: {response.url}")
                return True
            
        # Check response headers for blocking indicators
        headers = response.headers
        if b'cf-ray' in headers or b'cf-cache-status' in headers:
            self.logger.warning(f"CloudFlare protection detected on URL: {response.url}")
            return True
        
        return False

    def initial_parse(self, response):
        if self.check_if_blocked(response):
            return None
        
        # Extract links and follow them
        for rule in self._rules:
            links = rule.link_extractor.extract_links(response)
            for link in links:
                yield Request(
                    url=link.url,
                    callback=self.parse_house if rule.callback else None,
                    headers=response.request.headers,
                    meta=response.meta
                )

    def parse_house(self, response):
        # Check if we're being blocked
        if self.check_if_blocked(response):
            return

        item = FundaItem()
        item['url'] = response.url
        
        # Extract data from JSON-LD
        json_ld = None
        try:
            json_ld_script = response.css('script[type="application/ld+json"]::text').getall()
            for script in json_ld_script:
                data = json.loads(script)
                if isinstance(data, dict) and data.get('@type') in ['Product', 'Place', 'RealEstateListing', 'Appartement']:
                    json_ld = data
                    break
        except json.JSONDecodeError:
            self.logger.error(f"Failed to parse JSON-LD for URL: {response.url}")
        
        # Extract address
        if json_ld and 'address' in json_ld:
            address_data = json_ld['address']
            item['address'] = address_data.get('streetAddress', '')
            item['city'] = address_data.get('addressLocality', 'Amsterdam')
            item['postal_code'] = address_data.get('postalCode', '')
        else:
            # Fallback to breadcrumb
            breadcrumb_items = response.css('nav[aria-label="Breadcrumb"] span::text').getall()
            if breadcrumb_items:
                item['address'] = breadcrumb_items[-1].strip()
                item['city'] = 'Amsterdam'
                # Try to extract postal code from title
                title = response.css('title::text').get()
                if title:
                    postal_code_match = re.search(r'\b\d{4}\s?[A-Z]{2}\b', title)
                    if postal_code_match:
                        item['postal_code'] = postal_code_match.group(0)

        # Extract price
        if json_ld and 'offers' in json_ld and 'price' in json_ld['offers']:
            item['price'] = json_ld['offers']['price']
        else:
            # Try multiple price selectors
            price_selectors = [
                'dt:contains("Vraagprijs") + dd span::text',
                'dt:contains("Prijs") + dd span::text',
                'div[class*="price"] span::text',
                'span[class*="price"]::text'
            ]
            
            for selector in price_selectors:
                price_text = response.css(selector).get()
                if price_text:
                    # Extract numeric price
                    price_match = re.search(r'€\s*([\d.,]+)', price_text.replace('.', ''))
                    if price_match:
                        try:
                            price_str = price_match.group(1).replace(',', '')
                            item['price'] = int(float(price_str))
                            break
                        except ValueError:
                            continue

        # Extract property type
        if json_ld and '@type' in json_ld:
            item['property_type'] = json_ld['@type'].lower()
        else:
            property_type = response.css('dt:contains("Soort") + dd::text').get()
            if property_type:
                item['property_type'] = property_type.strip().lower()

        # Extract year built
        year_built = response.css('dt:contains("Bouwjaar") + dd::text').get()
        if year_built:
            try:
                item['year_built'] = int(year_built.strip())
            except ValueError:
                self.logger.warning(f"Could not parse year built: {year_built}")

        # Extract number of rooms
        rooms_text = response.css('dt:contains("Aantal kamers") + dd::text').get()
        if rooms_text:
            try:
                # Extract total rooms and bedrooms from text like "3 kamers (2 slaapkamers)"
                rooms_match = re.search(r'(\d+)\s+kamers?\s*(?:\((\d+)\s+slaapkamers?\))?', rooms_text)
                if rooms_match:
                    item['rooms'] = int(rooms_match.group(1))
                    if rooms_match.group(2):
                        item['bedrooms'] = int(rooms_match.group(2))
            except ValueError:
                self.logger.warning(f"Could not parse rooms: {rooms_text}")

        # Extract area (living area in m²)
        area_selectors = [
            'dt:contains("Woonoppervlakte") + dd::text',
            'dt:contains("Gebruiksoppervlakte wonen") + dd::text'
        ]
        for selector in area_selectors:
            area_text = response.css(selector).get()
            if area_text:
                try:
                    # Extract numeric area from text like "62 m²"
                    area_match = re.search(r'(\d+)\s*m²', area_text)
                    if area_match:
                        item['area'] = int(area_match.group(1))
                        break
                except ValueError:
                    continue

        self.logger.info(f"Successfully parsed {response.url}")
        self.logger.info(f"Extracted data: {item}")
        
        return item

import re
import scrapy
from scrapy.http import Request
from funda.core.items import FundaItem
import json
import random
from datetime import datetime
import urllib.parse

class FundaSpider(scrapy.Spider):
    name = "funda_spider"
    allowed_domains = ["funda.nl"]

    def __init__(self, place='amsterdam', max_pages=None, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.place = place
        # Set default max_pages to None if not specified
        self.max_pages = int(max_pages) if max_pages else None
        self.page_count = 1
        self.processed_urls = set()  # Track processed URLs
        self.total_items_scraped = 0
        
        # Base parameters for the search
        self.base_params = {
            'selected_area': json.dumps([place]),
            'availability': json.dumps(['available']),
            'object_type': json.dumps(['house', 'apartment']),
            'sort': 'date_down'
        }
        
        base_url = f"https://www.funda.nl/zoeken/koop/?{urllib.parse.urlencode(self.base_params)}"
        self.start_urls = [base_url]
        self.logger.info(f"Initial URL: {base_url}")
        self.logger.info(f"Maximum pages to scrape: {self.max_pages}")

        self.headers = {
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
            'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8',
            'Accept-Language': 'en-US,en;q=0.5',
            'Accept-Encoding': 'gzip, deflate, br',
            'Connection': 'keep-alive',
            'Cache-Control': 'no-cache',
            'Pragma': 'no-cache',
            'sec-ch-ua': '"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"',
            'sec-ch-ua-mobile': '?0',
            'sec-ch-ua-platform': '"macOS"'
        }

    def start_requests(self):
        for url in self.start_urls:
            yield Request(
                url=url,
                headers=self.headers,
                callback=self.parse,
                meta={'dont_cache': True}
            )

    def parse(self, response):
        self.logger.info(f"Parsing page {self.page_count}")
        
        # Check if we're being blocked or redirected
        if response.status in [403, 302, 503]:
            self.logger.error(f"Received status {response.status} for URL: {response.url}")
            return
        
        # Extract listings from both JSON-LD and HTML
        listing_urls = set()
        
        # 1. Try JSON-LD first
        json_ld_scripts = response.xpath('//script[@type="application/ld+json"]/text()').getall()
        for script in json_ld_scripts:
            try:
                data = json.loads(script)
                if isinstance(data, dict) and 'itemListElement' in data:
                    items = data['itemListElement']
                    for item in items:
                        url = item.get('url')
                        if url and '/detail/koop/' in url and url not in self.processed_urls:
                            listing_urls.add(url)
            except json.JSONDecodeError:
                continue
        
        # 2. Try HTML selectors as backup
        html_listings = response.css('div[data-test-id="search-result-item"] a::attr(href)').getall()
        html_listings.extend(response.css('div.search-result__header-title-col a::attr(href)').getall())
        
        for url in html_listings:
            if '/detail/koop/' in url and url not in self.processed_urls:
                full_url = response.urljoin(url)
                listing_urls.add(full_url)
        
        self.logger.info(f"Found {len(listing_urls)} new listings on page {self.page_count}")
        
        # Process found listings
        for url in listing_urls:
            self.processed_urls.add(url)
            yield scrapy.Request(
                url,
                callback=self.parse_house,
                headers=self.headers,
                meta={'dont_cache': True}
            )
        
        # Handle pagination if we haven't reached max_pages
        if not self.max_pages or self.page_count < self.max_pages:
            # Look for next page button
            next_page = response.css('a[data-test-id="next-page-button"]::attr(href)').get()
            if next_page:
                self.page_count += 1
                next_url = response.urljoin(next_page)
                self.logger.info(f"Moving to page {self.page_count}")
                yield scrapy.Request(
                    next_url,
                    callback=self.parse,
                    headers=self.headers,
                    meta={'dont_cache': True}
                )
            else:
                # Fallback to manual page construction if next button not found
                self.page_count += 1
                next_page_params = self.base_params.copy()
                next_page_params['page'] = self.page_count
                next_url = f"https://www.funda.nl/zoeken/koop/?{urllib.parse.urlencode(next_page_params)}"
                self.logger.info(f"Moving to page {self.page_count} (manual construction)")
                yield scrapy.Request(
                    next_url,
                    callback=self.parse,
                    headers=self.headers,
                    meta={'dont_cache': True}
                )
        else:
            self.logger.info(f"Reached maximum number of pages ({self.max_pages}). Stopping.")

    def parse_house(self, response):
        # Check if we're being blocked
        if response.status == 403:
            self.logger.error(f"Blocked (403) on URL: {response.url}")
            return

        item = FundaItem()
        item['url'] = response.url
        item['status'] = 'active'
        item['scraped_at'] = datetime.now().isoformat()
        
        # Extract property type and street from URL
        # Example URL: https://www.funda.nl/detail/koop/amsterdam/appartement-van-beuningenstraat-144-3/43801086/
        url_parts = response.url.split('/')
        self.logger.info(f"URL parts: {url_parts}")
        
        for i, part in enumerate(url_parts):
            if part.startswith('appartement-') or part.startswith('huis-'):
                # Extract property type
                property_type = 'appartement' if part.startswith('appartement-') else 'huis'
                item['property_type'] = property_type
                self.logger.info(f"Found property type: {property_type}")
                
                # Extract street address
                address_part = part[len(property_type)+1:].rsplit('-', 1)[0]
                self.logger.info(f"Raw address part: {address_part}")
                
                # Convert hyphens to spaces and capitalize
                street_parts = address_part.split('-')
                self.logger.info(f"Street parts: {street_parts}")
                
                # Combine number and any additions (like 102-1)
                if len(street_parts) >= 2:
                    street_name = ' '.join(street_parts[:-1])
                    number_part = street_parts[-1]
                    item['street'] = f"{street_name} {number_part}"
                else:
                    item['street'] = ' '.join(street_parts)
                
                self.logger.info(f"Extracted street: {item['street']}")
                break
            elif part.startswith('appartement') or part == 'huis':
                item['property_type'] = part
                self.logger.info(f"Found property type: {part}")
                
                # The next part should contain the street address
                if i + 1 < len(url_parts):
                    next_part = url_parts[i + 1]
                    address_part = next_part.rsplit('-', 1)[0]
                    self.logger.info(f"Raw address part: {address_part}")
                    
                    # Convert hyphens to spaces and capitalize
                    street_parts = address_part.split('-')
                    self.logger.info(f"Street parts: {street_parts}")
                    
                    # Combine number and any additions (like 102-1)
                    if len(street_parts) >= 2:
                        street_name = ' '.join(street_parts[:-1])
                        number_part = street_parts[-1]
                        item['street'] = f"{street_name} {number_part}"
                    else:
                        item['street'] = ' '.join(street_parts)
                    
                    self.logger.info(f"Extracted street: {item['street']}")
                break
        
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
        
        # Extract neighborhood and other address components
        if json_ld and 'address' in json_ld:
            address_data = json_ld['address']
            item['neighborhood'] = address_data.get('addressLocality', '').split(',')[0].strip()
            item['city'] = 'Amsterdam'
            item['postal_code'] = address_data.get('postalCode', '')
        else:
            # Fallback to breadcrumb
            breadcrumb_items = response.css('nav[aria-label="Breadcrumb"] span::text').getall()
            if breadcrumb_items:
                item['neighborhood'] = breadcrumb_items[-1].strip()
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
                # Extract total rooms from text like "3 kamers (2 slaapkamers)"
                rooms_match = re.search(r'(\d+)\s+kamers?', rooms_text)
                if rooms_match:
                    item['num_rooms'] = int(rooms_match.group(1))
            except ValueError:
                self.logger.warning(f"Could not parse rooms: {rooms_text}")

        # Extract area (living area in m²)
        area_selectors = [
            'dt:contains("Woonoppervlakte") + dd::text',
            'dt:contains("Gebruiksoppervlakte wonen") + dd::text',
            'li:contains("Woonoppervlakte") span.fd-text--emphasis::text',
            'li:contains("Gebruiksoppervlakte") span.fd-text--emphasis::text',
            'span:contains("m²")::text'
        ]
        
        for selector in area_selectors:
            area_text = response.css(selector).get()
            if area_text:
                self.logger.info(f"Found area text with selector '{selector}': {area_text}")
                try:
                    # Extract numeric area from text like "62 m²" or "62m²" or "62 m2"
                    area_match = re.search(r'(\d+)\s*(?:m²|m2)', area_text.strip())
                    if area_match:
                        item['living_area'] = int(area_match.group(1))
                        self.logger.info(f"Successfully extracted area: {item['living_area']} m²")
                        break
                except ValueError as e:
                    self.logger.warning(f"Failed to parse area from text '{area_text}': {e}")
                    continue

        # Extract listing date
        listing_date = response.css('dt:contains("Aangeboden sinds") + dd::text').get()
        if listing_date:
            try:
                # Convert Dutch month names to numbers
                dutch_months = {
                    'januari': '01', 'februari': '02', 'maart': '03', 'april': '04',
                    'mei': '05', 'juni': '06', 'juli': '07', 'augustus': '08',
                    'september': '09', 'oktober': '10', 'november': '11', 'december': '12'
                }
                
                # Clean and standardize the date text
                date_text = listing_date.lower().strip()
                for dutch, num in dutch_months.items():
                    date_text = date_text.replace(dutch, num)
                
                # Extract the date using regex
                date_match = re.search(r'(\d{1,2})\s+(\d{2})\s+(\d{4})', date_text)
                if date_match:
                    day, month, year = date_match.groups()
                    item['listing_date'] = f"{year}-{month}-{int(day):02d}"
            except Exception as e:
                self.logger.warning(f"Failed to parse listing date from text '{listing_date}': {e}")

        self.logger.info(f"Successfully parsed {response.url}")
        self.logger.info(f"Extracted data: {item}")
        
        return item

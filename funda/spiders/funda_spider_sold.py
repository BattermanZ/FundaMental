import re
import scrapy
from scrapy.spiders import CrawlSpider, Rule
from scrapy.linkextractors import LinkExtractor
from funda.core.items import FundaItem
from scrapy.http import Request
import json
import random
from datetime import datetime
import urllib.parse

class FundaSpiderSold(scrapy.Spider):
    name = "funda_spider_sold"
    allowed_domains = ["funda.nl"]
    
    def __init__(self, place='amsterdam', max_pages=None, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.place = place
        self.max_pages = int(max_pages) if max_pages else None
        self.page_count = 1
        self.processed_urls = set()  # Track processed URLs
        
        # Base parameters for the search
        self.base_params = {
            'selected_area': json.dumps([place]),
            'availability': json.dumps(['unavailable']),
            'object_type': json.dumps(['house', 'apartment']),
            'sort': 'date_down'
        }
        
        base_url = f"https://www.funda.nl/zoeken/koop/?{urllib.parse.urlencode(self.base_params)}"
        self.start_urls = [base_url]
        self.logger.info(f"Initial URL: {base_url}")

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
        self.logger.info(f"Parsing response from URL: {response.url}")
        
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
        
        self.logger.info(f"Found {len(listing_urls)} unique listings on page {self.page_count}")
        
        # Process found listings
        for url in listing_urls:
            self.processed_urls.add(url)
            yield scrapy.Request(
                url,
                callback=self.parse_listing,
                headers=self.headers,
                meta={'dont_cache': True}
            )
        
        # Handle pagination
        if self.max_pages is None or self.page_count < self.max_pages:
            # Look for next page button
            next_page = response.css('a[data-test-id="next-page-button"]::attr(href)').get()
            if next_page:
                self.page_count += 1
                next_url = response.urljoin(next_page)
                self.logger.info(f"Moving to next page: {next_url}")
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
                self.logger.info(f"Trying manual next page: {next_url}")
                yield scrapy.Request(
                    next_url,
                    callback=self.parse,
                    headers=self.headers,
                    meta={'dont_cache': True}
                )

    def parse_listing(self, response):
        self.logger.info(f"Parsing listing page: {response.url}")
        item = FundaItem()
        item['url'] = response.url

        # Find all JSON-LD scripts
        json_ld_scripts = response.css('script[type="application/ld+json"]::text').getall()
        self.logger.info(f"Found {len(json_ld_scripts)} JSON-LD scripts")

        # Extract dates from the page
        # First try to find dates in the JSON-LD data
        dates_found = False
        for script in json_ld_scripts:
            try:
                data = json.loads(script)
                if isinstance(data, dict):
                    self.logger.info(f"JSON-LD data type: {data.get('@type')}")
                    self.logger.info(f"JSON-LD keys: {data.keys()}")

                    # Check for dates in JSON-LD
                    if 'datePosted' in data:
                        item['listing_date'] = data['datePosted']
                        dates_found = True
                    if 'dateSold' in data:
                        item['selling_date'] = data['dateSold']
                        dates_found = True

                    # Extract basic info from JSON-LD
                    if data.get('@type') in ['Appartement', 'Product'] or (isinstance(data.get('@type'), list) and 'Appartement' in data['@type']):
                        item['street'] = data['address']['streetAddress']
                        item['city'] = data['address']['addressLocality']
                        item['postal_code'] = data['address'].get('postalCode', '')
                        item['price'] = data['offers']['price']

                        # Try to extract living area from JSON-LD first
                        if 'floorSize' in data:
                            try:
                                area_value = data['floorSize'].get('value', 0)
                                if area_value:
                                    item['living_area'] = int(float(str(area_value)))
                                    self.logger.info(f"Found living area in JSON-LD: {item['living_area']} m²")
                            except (ValueError, AttributeError) as e:
                                self.logger.warning(f"Could not parse living area from JSON-LD: {e}")
                        
                        # If living area not found in JSON-LD, try description and HTML
                        if not item.get('living_area'):
                            # Try to find area in description
                            description = data.get('description', '')
                            if description:
                                area_match = re.search(r'(\d+(?:[.,]\d+)?)\s*m²', description)
                                if area_match:
                                    try:
                                        area_str = area_match.group(1).replace(',', '.')
                                        item['living_area'] = int(float(area_str))
                                        self.logger.info(f"Found living area in description: {item['living_area']} m²")
                                    except (ValueError, AttributeError) as e:
                                        self.logger.warning(f"Could not parse living area from description: {e}")

                        # Try HTML selectors if still not found
                        if not item.get('living_area'):
                            area_selectors = [
                                'li:contains("m²") span.md\\:font-bold::text',
                                'dt:contains("Woonoppervlakte") + dd::text',
                                'span[data-testid="living-area"]::text',
                                'span[data-testid="floor-area"]::text',
                                'li:contains("Woonoppervlakte") span.fd-text--emphasis::text',
                                'li:contains("Gebruiksoppervlakte") span.fd-text--emphasis::text'
                            ]
                            
                            for selector in area_selectors:
                                area_text = response.css(selector).get()
                                if area_text:
                                    # Clean and extract numeric value
                                    match = re.search(r'(\d+(?:[.,]\d+)?)\s*(?:m²|m2)?', area_text)
                                    if match:
                                        try:
                                            # Replace comma with dot for decimal numbers
                                            area_str = match.group(1).replace(',', '.')
                                            item['living_area'] = int(float(area_str))
                                            self.logger.info(f"Found living area in HTML: {item['living_area']} m²")
                                            break
                                        except (ValueError, AttributeError) as e:
                                            self.logger.warning(f"Could not parse living area from HTML: {e}")

            except json.JSONDecodeError as e:
                self.logger.warning(f"Could not parse JSON-LD: {e}")

        # If dates not found in JSON-LD, try HTML selectors
        if not dates_found:
            # Try to find listing date and selling date in the HTML
            date_selectors = [
                'dt:contains("Aangeboden sinds") + dd::text',
                'dt:contains("Verkoopdatum") + dd::text',
                'li:contains("Aangeboden sinds") span.fd-text--emphasis::text',
                'li:contains("Verkoopdatum") span.fd-text--emphasis::text',
                'span[data-testid="listing-date"]::text',
                'span[data-testid="sale-date"]::text'
            ]
            
            for selector in date_selectors:
                date_text = response.css(selector).get()
                if date_text:
                    self.logger.info(f"Found date text with selector '{selector}': {date_text}")
                    try:
                        # Convert Dutch month names to numbers
                        dutch_months = {
                            'januari': '01', 'februari': '02', 'maart': '03', 'april': '04',
                            'mei': '05', 'juni': '06', 'juli': '07', 'augustus': '08',
                            'september': '09', 'oktober': '10', 'november': '11', 'december': '12'
                        }
                        
                        # Clean and standardize the date text
                        date_text = date_text.lower().strip()
                        for dutch, num in dutch_months.items():
                            date_text = date_text.replace(dutch, num)
                        
                        # Extract the date using regex
                        date_match = re.search(r'(\d{1,2})\s+(\d{2})\s+(\d{4})', date_text)
                        if date_match:
                            day, month, year = date_match.groups()
                            formatted_date = f"{year}-{month}-{int(day):02d}"
                            
                            if 'Aangeboden' in selector:
                                item['listing_date'] = formatted_date
                            elif 'Verkoop' in selector:
                                item['selling_date'] = formatted_date
                    except Exception as e:
                        self.logger.warning(f"Failed to parse date from text '{date_text}': {e}")

        # If address not found in JSON-LD, try HTML
        if not item.get('street') or not item.get('postal_code'):
            self.logger.info("Falling back to HTML parsing for address")
            
            # Try to get postal code and city from the text under the title
            address_text = response.css('h1.object-header__container span.text-neutral-40::text').get()
            if address_text:
                # Split postal code and city
                match = re.match(r'(\d{4}\s?[A-Z]{2})\s+(.+)', address_text)
                if match:
                    item['postal_code'] = match.group(1)
                    item['city'] = match.group(2)
                
                # Get street name
                street = response.css('h1.object-header__container span.block::text').get()
                if street:
                    item['street'] = street.strip()

        # Set the status and timestamp
        item['status'] = 'sold'
        item['scraped_at'] = datetime.now().isoformat()

        # Extract year built and number of rooms
        try:
            year_text = response.css('dt:contains("Bouwjaar") + dd::text').get()
            if year_text:
                self.logger.info(f"Found year_built: {year_text}")
                year_match = re.search(r'(\d{4})', year_text)
                if year_match:
                    item['year_built'] = int(year_match.group(1))
        except Exception as e:
            self.logger.warning(f"Could not parse year built: {e}")

        try:
            rooms_text = response.css('dt:contains("Aantal kamers") + dd::text').get()
            if rooms_text:
                self.logger.info(f"Found num_rooms: {rooms_text}")
                rooms_match = re.search(r'(\d+)\s*kamers?', rooms_text)
                if rooms_match:
                    item['num_rooms'] = int(rooms_match.group(1))
        except Exception as e:
            self.logger.warning(f"Could not parse number of rooms: {e}")

        self.logger.info(f"Extracted item: {item}")
        yield item

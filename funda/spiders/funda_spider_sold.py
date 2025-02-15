import re
import scrapy
from scrapy.spiders import CrawlSpider, Rule
from scrapy.linkextractors import LinkExtractor
from funda.items import FundaItem
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
        
        # Base parameters for the search
        self.base_params = {
            'selected_area': json.dumps([place]),  # JSON encode the array
            'availability': json.dumps(['unavailable']),  # JSON encode the array
            'object_type': json.dumps(['house', 'apartment']),  # JSON encode the array
            'sort': 'date_down'  # Sort by date descending
        }
        
        # Construct the base URL with encoded parameters
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
        
        # Log the response content for debugging
        self.logger.info(f"Response body preview: {response.text[:500]}")
        
        # Extract data from JSON-LD script
        json_ld_scripts = response.xpath('//script[@type="application/ld+json"]/text()').getall()
        self.logger.info(f"Found {len(json_ld_scripts)} JSON-LD scripts")
        
        found_listings = False
        for script in json_ld_scripts:
            try:
                data = json.loads(script)
                self.logger.info(f"JSON-LD data type: {type(data)}")
                self.logger.info(f"JSON-LD keys: {data.keys() if isinstance(data, dict) else 'not a dict'}")
                
                if isinstance(data, dict) and 'itemListElement' in data:
                    items = data['itemListElement']
                    self.logger.info(f"Found {len(items)} listings in JSON-LD data")
                    found_listings = True
                    
                    for item in items:
                        listing_url = item.get('url')
                        if listing_url and '/detail/koop/' in listing_url:
                            self.logger.info(f"Found listing URL: {listing_url}")
                            yield scrapy.Request(
                                listing_url,
                                callback=self.parse_listing,
                                headers=response.request.headers,
                                meta={'dont_cache': True}
                            )
            except json.JSONDecodeError as e:
                self.logger.error(f"Failed to parse JSON-LD data: {e}")
                self.logger.error(f"Problematic JSON-LD content: {script[:200]}")
                continue
        
        if not found_listings:
            self.logger.warning(f"No listings found in JSON-LD data for URL: {response.url}")
            self.logger.warning("Response headers:")
            for header, value in response.headers.items():
                self.logger.warning(f"{header}: {value}")

        # Check if we should proceed to the next page
        if self.max_pages is None or self.page_count < self.max_pages:
            self.page_count += 1
            next_page_params = self.base_params.copy()
            next_page_params['page'] = self.page_count
            next_page_url = f"https://www.funda.nl/zoeken/koop/?{urllib.parse.urlencode(next_page_params)}"
            self.logger.info(f"Moving to next page: {next_page_url}")
            yield scrapy.Request(
                next_page_url,
                callback=self.parse,
                headers=response.request.headers,
                meta={'dont_cache': True}
            )

    def parse_listing(self, response):
        self.logger.info(f"Parsing listing page: {response.url}")
        
        # Find all JSON-LD scripts
        json_ld_scripts = response.xpath('//script[@type="application/ld+json"]/text()').getall()
        self.logger.info(f"Found {len(json_ld_scripts)} JSON-LD scripts")
        
        item = {}
        item['url'] = response.url
        
        # Try to extract data from JSON-LD first
        for script in json_ld_scripts:
            try:
                data = json.loads(script)
                self.logger.info(f"JSON-LD data type: {data.get('@type')}")
                self.logger.info(f"JSON-LD keys: {list(data.keys())}")
                
                if 'address' in data:
                    item['street'] = data['address'].get('streetAddress', '')
                    item['city'] = data['address'].get('addressLocality', '')
                    item['postal_code'] = data['address'].get('postalCode', '')
                    
                    # Try to extract postal code from description if not in address
                    if not item['postal_code'] and 'description' in data:
                        postal_match = re.search(r'(\d{4}\s?[A-Z]{2})', data['description'])
                        if postal_match:
                            item['postal_code'] = postal_match.group(1)
                
                if 'offers' in data and 'price' in data['offers']:
                    item['price'] = data['offers']['price']
                    
            except json.JSONDecodeError as e:
                self.logger.error(f"Error parsing JSON-LD: {e}")
                self.logger.error(f"Problematic content: {script[:200]}...")
        
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
            
            # Try alternative postal code selector
            if not item.get('postal_code'):
                postal_text = response.css('span[data-testid="postal-code"]::text').get()
                if postal_text:
                    postal_match = re.search(r'(\d{4}\s?[A-Z]{2})', postal_text)
                    if postal_match:
                        item['postal_code'] = postal_match.group(1)
        
        # Set status and timestamp
        item['status'] = 'sold'
        item['timestamp'] = datetime.now().isoformat()
        
        # Extract other details
        selectors = {
            'year_built': [
                'dt:contains("Construction period") + dd::text',
                'dt:contains("Bouwjaar") + dd::text',
                'dt:contains("Bouwperiode") + dd::text'
            ],
            'living_area': [
                'dt:contains("Living area") + dd::text',
                'dt:contains("Woonoppervlakte") + dd::text',
                'dt:contains("Gebruiksoppervlakte wonen") + dd::text'
            ],
            'num_rooms': [
                'dt:contains("Number of rooms") + dd::text',
                'dt:contains("Aantal kamers") + dd::text',
                'dt:contains("Kamers") + dd::text'
            ]
        }
        
        for field, selector_list in selectors.items():
            for selector in selector_list:
                value = response.css(selector).get()
                if value:
                    item[field] = value.strip()
                    self.logger.info(f"Found {field}: {value.strip()}")
                    break
        
        # Clean up numeric fields
        if item.get('year_built'):
            try:
                item['year_built'] = int(re.search(r'\d+', str(item['year_built'])).group())
            except (ValueError, AttributeError):
                item['year_built'] = None
            
        if item.get('living_area'):
            try:
                item['living_area'] = int(re.search(r'\d+', str(item['living_area'])).group())
            except (ValueError, AttributeError):
                item['living_area'] = None
            
        if item.get('num_rooms'):
            try:
                item['num_rooms'] = int(re.search(r'\d+', str(item['num_rooms'])).group())
            except (ValueError, AttributeError):
                item['num_rooms'] = None
        
        self.logger.info(f"Extracted item: {item}")
        yield item

import re
import scrapy
from scrapy.http import Request
from spiders.items import FundaItem
import json
from datetime import datetime
import urllib.parse
import os
import pickle
import logging

class FundaSpider(scrapy.Spider):
    name = "funda_spider"
    allowed_domains = ["funda.nl"]
    
    # Only override settings that are specific to this spider
    custom_settings = {
        'HTTPCACHE_ENABLED': False  # Disable caching for active listings to ensure freshness'
    }

    def __init__(self, place='amsterdam', max_pages=None, *args, **kwargs):
        super(FundaSpider, self).__init__(*args, **kwargs)
        self.place = place.lower()  # Ensure lowercase for consistency
        self.original_city = kwargs.get('original_city', place)
        self.max_pages = int(max_pages) if max_pages else None
        self.page_count = 1
        self.processed_urls = set()  # Track processed URLs in current run
        self.total_items_scraped = 0
        self.new_items_found = 0  # Track new items found
        self.active_urls = set()  # Track all active URLs for refresh operation
        self.buffer = []
        self.buffer_size = 100  # Configurable batch size
        self.logger = logging.getLogger(__name__)
        
        # Create state directory if it doesn't exist
        self.state_dir = os.path.join(os.getcwd(), '.spider_state')
        os.makedirs(self.state_dir, exist_ok=True)
        self.state_file = os.path.join(self.state_dir, f'funda_active_{self.place}_state.pkl')
        
        # Log city information
        self.logger.info(f"Spider initialized for city: {self.original_city} (normalized: {self.place})")
        
        # Base parameters for the search
        self.base_params = {
            'selected_area': json.dumps([self.place]),
            'availability': json.dumps(['available']),
            'object_type': json.dumps(['house', 'apartment']),
            'sort': 'date_down'  # Most recent first
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

        self.state = {}
        self.load_state()

    def save_state(self):
        """Save current spider state for resuming later."""
        state = {
            'page_count': self.page_count,
            'processed_urls': self.processed_urls,
            'total_items_scraped': self.total_items_scraped,
            'new_items_found': self.new_items_found
        }
        with open(self.state_file, 'wb') as f:
            pickle.dump(state, f)
        self.logger.info(f"Saved state: Page {self.page_count}, Items {self.total_items_scraped}, New Items {self.new_items_found}")

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
        
        # Extract all listing URLs from the page first
        all_listing_urls = set()
        
        # 1. Try JSON-LD first
        json_ld_scripts = response.xpath('//script[@type="application/ld+json"]/text()').getall()
        for script in json_ld_scripts:
            try:
                data = json.loads(script)
                if isinstance(data, dict) and 'itemListElement' in data:
                    items = data['itemListElement']
                    for item in items:
                        url = item.get('url')
                        if url and '/detail/koop/' in url:
                            all_listing_urls.add(url)
            except json.JSONDecodeError:
                continue
        
        # 2. Try HTML selectors as backup
        html_listings = response.css('div[data-test-id="search-result-item"] a::attr(href)').getall()
        html_listings.extend(response.css('div.search-result__header-title-col a::attr(href)').getall())
        
        for url in html_listings:
            if '/detail/koop/' in url:
                full_url = response.urljoin(url)
                all_listing_urls.add(full_url)

        # Now filter out existing URLs
        new_listing_urls = {url for url in all_listing_urls 
                          if url not in self.processed_urls}
        
        # If we found no new listings on this page, stop crawling
        if not new_listing_urls:
            self.logger.info(f"No new listings found on page {self.page_count}, all URLs already exist in database. Stopping crawl.")
            return

        # Log stats about new vs existing URLs
        self.logger.info(f"Found {len(all_listing_urls)} total listings on page {self.page_count}")
        self.logger.info(f"Found {len(new_listing_urls)} new listings to process")
        self.new_items_found += len(new_listing_urls)
        
        # Process only new listings
        for url in new_listing_urls:
            self.processed_urls.add(url)
            yield scrapy.Request(
                url,
                callback=self.parse_property,
                headers=self.headers,
                meta={'dont_cache': True}
            )
        
        # Save state after processing each page
        self.save_state()
        
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
                # Fallback to manual page construction if the next button is not found
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

    def parse_property(self, response):
        # Check if we're being blocked
        if response.status == 403 or "Je bent bijna op de pagina die je zoekt" in response.text:
            self.logger.error(f"Blocked or verification required for URL: {response.url}")
            return

        item = self.extract_property_data(response)
        if item:
            self.total_items_scraped += 1
            self.buffer.append(item)
            
            # Log progress every 10 items
            if self.total_items_scraped % 10 == 0:
                self.logger.info(f"Scraped {self.total_items_scraped} properties. Buffer size: {len(self.buffer)}")
            
            # If buffer is full, yield the batch
            if len(self.buffer) >= self.buffer_size:
                self.logger.info(f"Buffer full ({len(self.buffer)} items). Yielding batch...")
                batch = {
                    'type': 'properties_batch',
                    'items': self.buffer.copy(),
                    'timestamp': datetime.now().isoformat(),
                    'spider': self.name,
                    'city': self.place
                }
                self.buffer = []
                yield batch

    def extract_property_data(self, response):
        # Check if we're being blocked
        if response.status == 403 or "Je bent bijna op de pagina die je zoekt" in response.text:
            self.logger.error(f"Blocked or verification required for URL: {response.url}")
            return

        # Initialize item with status 'active' by default
        item = FundaItem(url=response.url, status='active')
        
        # Extract address from the page content
        # Try multiple selectors for the address
        address_selectors = [
            'h1.object-header__title::text',  # Main title
            'h1.object-header__container span.object-header__title::text',  # New title format
            'h1.object-header__container span.object-header__street::text',  # Street only
            'h1.object-header__container span.object-header__house-number::text',  # House number
            'div.object-header__details h1 span::text',  # Alternative format
            'div.object-header__details-info h1.fd-m-none::text',  # Another format
        ]
        
        # First try to get street and house number separately
        street = response.css('h1.object-header__container span.object-header__street::text').get()
        house_number = response.css('h1.object-header__container span.object-header__house-number::text').get()
        
        if street and house_number:
            item['street'] = f"{street.strip()} {house_number.strip()}"
            self.logger.info(f"Extracted street from separate components: {item['street']}")
        else:
            # Try to get full address from other selectors
            for selector in address_selectors:
                address = response.css(selector).get()
                if address:
                    address = address.strip()
                    self.logger.info(f"Found address with selector '{selector}': {address}")
                    # Extract street name and number
                    # Pattern matches street name followed by number, handling various formats
                    match = re.match(r'^(.*?)\s*(\d+(?:\s*[a-zA-Z-]?\d*)?)\s*$', address)
                    if match:
                        street_name, number = match.groups()
                        item['street'] = f"{street_name.strip()} {number.strip()}"
                        self.logger.info(f"Extracted street: {item['street']}")
                        break

        # If we still don't have a street address, try JSON-LD
        if not item.get('street'):
            try:
                json_ld_script = response.css('script[type="application/ld+json"]::text').getall()
                for script in json_ld_script:
                    data = json.loads(script)
                    if isinstance(data, dict) and 'address' in data:
                        street_address = data['address'].get('streetAddress')
                        if street_address:
                            item['street'] = street_address.strip()
                            self.logger.info(f"Extracted street from JSON-LD: {item['street']}")
                            break
            except (json.JSONDecodeError, KeyError, AttributeError) as e:
                self.logger.warning(f"Failed to extract address from JSON-LD: {e}")

        # Extract property type from breadcrumbs or JSON-LD
        property_type_selectors = [
            'nav[aria-label="Breadcrumb"] span:contains("appartement")::text',
            'nav[aria-label="Breadcrumb"] span:contains("huis")::text'
        ]
        
        for selector in property_type_selectors:
            property_type = response.css(selector).get()
            if property_type:
                item['property_type'] = property_type.strip().lower()
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
            item['city'] = self.place.capitalize()
            item['postal_code'] = address_data.get('postalCode', '')
        else:
            # Fallback to breadcrumb
            breadcrumb_items = response.css('nav[aria-label="Breadcrumb"] span::text').getall()
            if breadcrumb_items:
                item['neighborhood'] = breadcrumb_items[-1].strip()
                item['city'] = self.place.capitalize()
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

        # Extract energy label from Kenmerken section first
        energy_label_selectors = [
            'dt:contains("Energielabel") + dd span::text',  # New format with span
            'dt:contains("Energielabel") + dd div span::text',  # Alternative format
            'dt:contains("Energielabel") + dd::text',  # Old format
            'span[data-test-id="energy-label"]::text',
            'span[class*="energy-label"]::text'
        ]
        
        for selector in energy_label_selectors:
            energy_label = response.css(selector).get()
            if energy_label:
                clean_label = energy_label.strip().upper()
                if re.match(r'^[A-G](\+{1,2})?$', clean_label):
                    item['energy_label'] = clean_label
                    self.logger.info(f"Found energy label with selector '{selector}': {item['energy_label']}")
                    break

        # If not found in selectors, try JSON-LD
        if not item.get('energy_label'):
            try:
                json_ld_scripts = response.css('script[type="application/ld+json"]::text').getall()
                for script in json_ld_scripts:
                    data = json.loads(script)
                    if isinstance(data, dict):
                        # Try to find energy label in the JSON-LD data
                        if 'EnergyData' in str(data) or 'energyLabel' in str(data):
                            energy_match = re.search(r'["\']energy(?:Label|Data)["\']\s*:\s*["\']([A-G]\+*)["\']', script, re.IGNORECASE)
                            if energy_match:
                                item['energy_label'] = energy_match.group(1).upper()
                                self.logger.info(f"Found energy label in JSON-LD: {item['energy_label']}")
                                break
            except (json.JSONDecodeError, AttributeError) as e:
                self.logger.warning(f"Failed to extract energy label from JSON-LD: {e}")

        # If still not found, try description as fallback
        if not item.get('energy_label'):
            description = response.css('div.object-description__features li::text, div.object-description-body *::text').getall()
            for text in description:
                text = text.strip().lower()
                if 'energielabel' in text or 'energieklasse' in text:
                    # More flexible pattern matching
                    label_match = re.search(r'energi(?:elabel|eklasse)\s*([a-g](?:\+{1,2})?)', text)
                    if label_match:
                        item['energy_label'] = label_match.group(1).upper()
                        self.logger.info(f"Found energy label in description: {item['energy_label']}")
                        break

        if not item.get('energy_label'):
            self.logger.warning("Could not find energy label")

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

        # Add scraped timestamp
        item['scraped_at'] = datetime.utcnow().isoformat()

        if self.total_items_scraped % 10 == 0:  # Log progress every 10 items
            self.logger.info(f"Progress: Scraped {self.total_items_scraped} items from {self.page_count}")
        
        self.logger.info(f"Successfully parsed {response.url}")
        self.logger.info(f"Extracted data: {item}")
        
        return item

    def collect_active_urls(self, response):
        """
        Collects only URLs from listing pages without visiting individual properties.
        Used for the weekly refresh operation.
        """
        self.logger.info(f"Collecting URLs from page {self.page_count}")
        
        # Extract all listing URLs from the page
        all_listing_urls = set()
        
        # 1. Try JSON-LD first
        json_ld_scripts = response.xpath('//script[@type="application/ld+json"]/text()').getall()
        for script in json_ld_scripts:
            try:
                data = json.loads(script)
                if isinstance(data, dict) and 'itemListElement' in data:
                    items = data['itemListElement']
                    for item in items:
                        url = item.get('url')
                        if url and '/koop/' in url:
                            all_listing_urls.add(url)
            except json.JSONDecodeError:
                continue
        
        # 2. Try HTML selectors as backup
        html_listings = response.css('div[data-test-id="search-result-item"] a::attr(href)').getall()
        html_listings.extend(response.css('div.search-result__header-title-col a::attr(href)').getall())
        
        for url in html_listings:
            if '/koop/' in url:
                full_url = response.urljoin(url)
                all_listing_urls.add(full_url)

        # Add all found URLs to the active_urls set
        self.active_urls.update(all_listing_urls)
        
        # Handle pagination
        if self.max_pages is None or self.page_count < self.max_pages:
            next_page = response.css('a[data-test-id="next-page-button"]::attr(href)').get()
            if next_page:
                self.page_count += 1
                next_url = response.urljoin(next_page)
                self.logger.info(f"Moving to page {self.page_count}")
                yield scrapy.Request(
                    next_url,
                    callback=self.collect_active_urls,
                    headers=self.headers,
                    meta={'dont_cache': True}
                )

    def refresh_active_listings(self):
        """
        Special method to only collect URLs for the refresh operation.
        Returns the set of all active URLs found.
        """
        self.active_urls.clear()
        self.page_count = 1
        
        # Construct the URL for active listings
        params = {
            'selected_area': json.dumps([self.place]),
            'availability': json.dumps(['available']),
            'object_type': json.dumps(['house', 'apartment']),
            'sort': 'date_down'
        }
        
        url = f"https://www.funda.nl/zoeken/koop/?{urllib.parse.urlencode(params)}"
        
        return scrapy.Request(
            url=url,
            headers=self.headers,
            callback=self.collect_active_urls,
            meta={'dont_cache': True}
        )

    def closed(self, reason):
        """Called when the spider is closed."""
        self.logger.info(f"Spider closed: {reason}")
        self.logger.info(f"Final statistics:")
        self.logger.info(f"Total pages scraped: {self.page_count}")
        self.logger.info(f"Total new items found: {self.new_items_found}")
        self.logger.info(f"Total items scraped: {self.total_items_scraped}")
        self.logger.info(f"Total unique URLs processed: {len(self.processed_urls)}")
        
        # Save final state
        self.save_state()
        
        # Ensure any remaining items in buffer are processed when spider closes
        if self.buffer:
            self.logger.info(f"Flushing remaining {len(self.buffer)} properties on spider close")
            batch = {
                'type': 'properties_batch',
                'items': self.buffer.copy(),
                'timestamp': datetime.now().isoformat(),
                'spider': self.name,
                'city': self.place
            }
            self.buffer = []
            yield batch 
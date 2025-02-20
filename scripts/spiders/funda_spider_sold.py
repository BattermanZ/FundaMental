import re
import scrapy
from scrapy.http import Request
from spiders.items import FundaItem
import json
from datetime import datetime
import urllib.parse
import os
import pickle

class FundaSpiderSold(scrapy.Spider):
    name = "funda_spider_sold"
    allowed_domains = ["funda.nl"]
    
    # Only override settings that are specific to this spider
    custom_settings = {
        'HTTPCACHE_ENABLED': True  # Enable caching for sold listings as they don't change
    }

    def __init__(self, place='amsterdam', max_pages=None, resume=False, *args, **kwargs):
        super(FundaSpiderSold, self).__init__(*args, **kwargs)
        self.place = place.lower()  # Ensure lowercase for consistency
        self.original_city = kwargs.get('original_city', place)
        self.max_pages = int(max_pages) if max_pages else None
        self.page_count = 1
        self.processed_urls = set()  # Track processed URLs in current run
        self.total_items_scraped = 0
        self.new_items_found = 0  # Track new items found
        self.resume = resume
        self.state = {}
        self.load_state() if resume else {}
        self.buffer = []
        self.buffer_size = 100  # Configurable batch size
        
        # Create state directory if it doesn't exist
        self.state_dir = os.path.join(os.getcwd(), '.spider_state')
        os.makedirs(self.state_dir, exist_ok=True)
        self.state_file = os.path.join(self.state_dir, f'funda_sold_{self.place}_state.pkl')
        
        # Log city information
        self.logger.info(f"Spider initialized for city: {self.original_city} (normalized: {self.place})")
        
        # Base parameters for the search
        self.base_params = {
            'selected_area': json.dumps([self.place]),
            'availability': json.dumps(['unavailable']),
            'object_type': json.dumps(['house', 'apartment']),
            'sort': 'date_down'  # Most recent first
        }
        
        # If resuming and we have a page count, start from there
        if self.resume and self.page_count > 1:
            self.base_params['page'] = self.page_count
            self.logger.info(f"Resuming from page {self.page_count}")
        
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

    def load_state(self):
        """Load previous spider state."""
        try:
            with open(self.state_file, 'rb') as f:
                state = pickle.load(f)
                self.page_count = state['page_count']
                self.processed_urls = state['processed_urls']
                self.total_items_scraped = state['total_items_scraped']
                self.new_items_found = state.get('new_items_found', 0)  # Backward compatibility
                self.logger.info(f"Loaded state: Page {self.page_count}, Items {self.total_items_scraped}, New Items {self.new_items_found}")
        except Exception as e:
            self.logger.error(f"Error loading state: {e}")

    def start_requests(self):
        for url in self.start_urls:
            yield Request(
                url=url,
                headers=self.headers,
                callback=self.parse,
                meta={'dont_cache': True}
            )

    def parse(self, response):
        self.logger.info(f"Parsing page {self.page_count} of max {self.max_pages}")
        
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
                callback=self.parse_listing,
                headers=self.headers,
                meta={'dont_cache': True}
            )
        
        # Save state after processing each page
        self.save_state()
        
        # Handle pagination if we haven't reached max_pages
        if self.page_count < self.max_pages:
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

    def parse_listing(self, response):
        self.logger.info(f"Parsing listing page: {response.url}")
        
        # Check if we're being blocked
        if response.status == 403 or "Je bent bijna op de pagina die je zoekt" in response.text:
            self.logger.error(f"Blocked or verification required for URL: {response.url}")
            return

        item = self.extract_property_data(response)
        if item:
            self.total_items_scraped += 1
            self.buffer.append(item)
            
            # If buffer is full, yield the batch
            if len(self.buffer) >= self.buffer_size:
                yield from self.flush_buffer()

    def extract_property_data(self, response):
        item = FundaItem()
        item['url'] = response.url
        item['status'] = 'sold'
        
        # Extract energy label using documented approach
        energy_label_selectors = [
            'dt:contains("Energielabel") + dd span::text',  # New format with span
            'dt:contains("Energielabel") + dd div span::text',  # Alternative format
            'dt:contains("Energielabel") + dd::text',  # Old format
            'span[data-test-id="energy-label"]::text',
            'span[class*="energy-label"]::text'
        ]
        
        # Try HTML selectors first
        for selector in energy_label_selectors:
            energy_label = response.css(selector).get()
            if energy_label:
                clean_label = energy_label.strip().upper()
                if re.match(r'^[A-G](\+{1,2})?$', clean_label):
                    item['energy_label'] = clean_label
                    self.logger.info(f"Found energy label with selector '{selector}': {item['energy_label']}")
                    break
        
        # Find all JSON-LD scripts
        json_ld_scripts = response.css('script[type="application/ld+json"]::text').getall()
        self.logger.info(f"Found {len(json_ld_scripts)} JSON-LD scripts")

        # If energy label not found in HTML, try JSON-LD
        if not item.get('energy_label'):
            try:
                for script in json_ld_scripts:
                    data = json.loads(script)
                    if isinstance(data, dict):
                        if 'EnergyData' in str(data) or 'energyLabel' in str(data):
                            energy_match = re.search(r'["\']energy(?:Label|Data)["\']\s*:\s*["\']([A-G]\+*)["\']', script, re.IGNORECASE)
                            if energy_match:
                                item['energy_label'] = energy_match.group(1).upper()
                                self.logger.info(f"Found energy label in JSON-LD: {item['energy_label']}")
                                break
            except (json.JSONDecodeError, AttributeError) as e:
                self.logger.warning(f"Failed to extract energy label from JSON-LD: {e}")

        # If still not found, try description text
        if not item.get('energy_label'):
            description = response.css('div.object-description__features li::text, div.object-description-body *::text').getall()
            for text in description:
                text = text.strip().lower()
                if 'energielabel' in text or 'energieklasse' in text:
                    label_match = re.search(r'energi(?:elabel|eklasse)\s*([a-g](?:\+{1,2})?)', text)
                    if label_match:
                        item['energy_label'] = label_match.group(1).upper()
                        self.logger.info(f"Found energy label in description: {item['energy_label']}")
                        break

        if not item.get('energy_label'):
            self.logger.warning("Could not find energy label")

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
                        if 'address' in data:
                            item['street'] = data['address'].get('streetAddress')
                            item['city'] = self.place.capitalize()
                            item['postal_code'] = data['address'].get('postalCode')
                        if 'offers' in data and 'price' in data['offers']:
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

        # Extract area (living area in m²)
        if not item.get('living_area'):
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

        self.logger.info(f"Extracted item: {item}")
        return item

    def flush_buffer(self):
        """Flush the current buffer of properties."""
        if not self.buffer:
            return
            
        self.logger.info(f"Flushing buffer with {len(self.buffer)} properties")
        properties_batch = self.buffer
        self.buffer = []
        
        yield {
            'type': 'properties_batch',
            'items': properties_batch,
            'timestamp': datetime.now().isoformat(),
            'spider': self.name,
            'city': self.place
        }

    def closed(self, reason):
        """Called when the spider is closed."""
        self.logger.info(f"Spider closed: {reason}")
        self.logger.info(f"Final statistics:")
        self.logger.info(f"Total pages scraped: {self.page_count}")
        self.logger.info(f"Total new items found: {self.new_items_found}")
        self.logger.info(f"Total items scraped: {self.total_items_scraped}")
        self.logger.info(f"Total unique URLs processed: {len(self.processed_urls)}")
        
        # Flush any remaining items in buffer
        if self.buffer:
            self.logger.info(f"Flushing remaining {len(self.buffer)} properties on spider close")
            yield from self.flush_buffer()
        
        # Save final state
        self.save_state() 
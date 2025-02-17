import re
import scrapy
from scrapy.http import Request
from scrapers.funda.items import FundaItem
from scrapers.funda.database import FundaDB  # Import the database module
import json
from datetime import datetime
import urllib.parse
import os
import pickle

class FundaSpiderSold(scrapy.Spider):
    name = "funda_spider_sold"
    allowed_domains = ["funda.nl"]
    
    custom_settings = {
        'DOWNLOAD_DELAY': 2,
        'CONCURRENT_REQUESTS': 2,
        'CONCURRENT_REQUESTS_PER_DOMAIN': 2,
        'AUTOTHROTTLE_ENABLED': True,
        'AUTOTHROTTLE_START_DELAY': 2,
        'AUTOTHROTTLE_MAX_DELAY': 30,
        'AUTOTHROTTLE_TARGET_CONCURRENCY': 2.0,
        'DOWNLOAD_TIMEOUT': 30,
        'ITEM_PIPELINES': {
            'scrapers.funda.pipelines.FundaPipeline': 300,
            'scrapers.funda.pipelines.JsonMessagePipeline': 900,
        }
    }

    def __init__(self, place='amsterdam', max_pages=None, resume=False, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.place = place
        self.max_pages = int(max_pages) if max_pages else None  # Remove default of 200 pages
        self.page_count = 1
        self.processed_urls = set()  # Track processed URLs in current run
        self.total_items_scraped = 0
        self.new_items_found = 0  # Track new items found
        self.resume = resume
        
        # Initialize database connection for URL checking
        self.db = FundaDB()
        self.existing_urls = self.db.get_existing_urls()  # Get existing URLs from DB
        self.logger.info(f"Found {len(self.existing_urls)} existing URLs in database")
        
        # Create state directory if it doesn't exist
        self.state_dir = os.path.join(os.getcwd(), '.spider_state')
        os.makedirs(self.state_dir, exist_ok=True)
        self.state_file = os.path.join(self.state_dir, f'funda_sold_{place}_state.pkl')
        
        # Load previous state if resuming
        if self.resume and os.path.exists(self.state_file):
            self.load_state()
        
        # Base parameters for the search
        self.base_params = {
            'selected_area': json.dumps([place]),
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
                          if url not in self.processed_urls and url not in self.existing_urls}
        
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

        item = FundaItem(url=response.url, status='sold')
        
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
                        item.listing_date = data['datePosted']
                        dates_found = True
                    if 'dateSold' in data:
                        item.selling_date = data['dateSold']
                        dates_found = True

                    # Extract basic info from JSON-LD
                    if data.get('@type') in ['Appartement', 'Product'] or (isinstance(data.get('@type'), list) and 'Appartement' in data['@type']):
                        if 'address' in data:
                            item.street = data['address'].get('streetAddress')
                            item.city = data['address'].get('addressLocality')
                            item.postal_code = data['address'].get('postalCode')
                        if 'offers' in data and 'price' in data['offers']:
                            item.price = data['offers']['price']

                        # Try to extract living area from JSON-LD first
                        if 'floorSize' in data:
                            try:
                                area_value = data['floorSize'].get('value', 0)
                                if area_value:
                                    item.living_area = int(float(str(area_value)))
                                    self.logger.info(f"Found living area in JSON-LD: {item.living_area} m²")
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
                                item.listing_date = formatted_date
                            elif 'Verkoop' in selector:
                                item.selling_date = formatted_date
                    except Exception as e:
                        self.logger.warning(f"Failed to parse date from text '{date_text}': {e}")

        # If address not found in JSON-LD, try HTML
        if not item.street or not item.postal_code:
            self.logger.info("Falling back to HTML parsing for address")
            
            # Try to get postal code and city from the text under the title
            address_text = response.css('h1.object-header__container span.text-neutral-40::text').get()
            if address_text:
                # Split postal code and city
                match = re.match(r'(\d{4}\s?[A-Z]{2})\s+(.+)', address_text)
                if match:
                    item.postal_code = match.group(1)
                    item.city = match.group(2)
                
                # Get street name
                street = response.css('h1.object-header__container span.block::text').get()
                if street:
                    item.street = street.strip()

        # Extract year built and number of rooms
        try:
            year_text = response.css('dt:contains("Bouwjaar") + dd::text').get()
            if year_text:
                self.logger.info(f"Found year_built: {year_text}")
                year_match = re.search(r'(\d{4})', year_text)
                if year_match:
                    item.year_built = int(year_match.group(1))
        except Exception as e:
            self.logger.warning(f"Could not parse year built: {e}")

        try:
            rooms_text = response.css('dt:contains("Aantal kamers") + dd::text').get()
            if rooms_text:
                self.logger.info(f"Found num_rooms: {rooms_text}")
                rooms_match = re.search(r'(\d+)\s*kamers?', rooms_text)
                if rooms_match:
                    item.num_rooms = int(rooms_match.group(1))
        except Exception as e:
            self.logger.warning(f"Could not parse number of rooms: {e}")

        # Extract area (living area in m²)
        if not item.living_area:
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
                            item.living_area = int(area_match.group(1))
                            self.logger.info(f"Successfully extracted area: {item.living_area} m²")
                            break
                    except ValueError as e:
                        self.logger.warning(f"Failed to parse area from text '{area_text}': {e}")
                        continue

        self.total_items_scraped += 1
        if self.total_items_scraped % 10 == 0:  # Log progress every 10 items
            self.logger.info(f"Progress: Scraped {self.total_items_scraped} items from {self.page_count} pages")
        
        self.logger.info(f"Extracted item: {item}")
        return item

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
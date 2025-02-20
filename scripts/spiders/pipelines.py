import json
from datetime import datetime
import os
import requests
import logging
import time
from scrapy.exceptions import DropItem

class TestPipeline:
    def process_item(self, item, spider):
        print("\nScraped Item:")
        for key, value in item.items():
            print(f"{key}: {value}")
        print("-" * 50)
        return item

class JsonExportPipeline:
    def open_spider(self, spider):
        # Create output directory if it doesn't exist
        os.makedirs('output', exist_ok=True)
        # Create a filename with timestamp
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        filename = f'output/{spider.name}_{timestamp}.json'
        self.file = open(filename, 'w', encoding='utf-8')
        self.file.write('[\n')  # Start JSON array
        self.first_item = True

    def close_spider(self, spider):
        self.file.write('\n]')  # End JSON array
        self.file.close()

    def process_item(self, item, spider):
        line = json.dumps(dict(item), ensure_ascii=False, indent=2)
        if not self.first_item:
            self.file.write(',\n')
        self.file.write(line)
        self.first_item = False
        return item

class FundaPipeline:
    def __init__(self):
        self.api_url = os.getenv('API_URL', 'http://localhost:5250')
        self.session = requests.Session()
        self.logger = logging.getLogger(__name__)
        self.retry_count = 3
        self.retry_delay = 5  # seconds

    def process_item(self, item, spider):
        """Process either a batch of items or a single item."""
        if isinstance(item, dict) and item.get('type') == 'properties_batch':
            return self.process_batch(item, spider)
        return self.process_single_item(item, spider)

    def process_batch(self, batch_item, spider):
        """Process a batch of properties."""
        properties = batch_item.get('items', [])
        if not properties:
            return batch_item

        for attempt in range(self.retry_count):
            try:
                response = self.session.post(
                    f"{self.api_url}/api/properties/batch",
                    json={
                        'properties': [dict(item) for item in properties],
                        'spider': batch_item.get('spider', spider.name if spider else 'unknown'),
                        'city': batch_item.get('city', spider.place if spider else 'unknown')
                    },
                    timeout=30
                )
                response.raise_for_status()
                self.logger.info(f"Successfully processed batch of {len(properties)} properties")
                return batch_item

            except requests.exceptions.RequestException as e:
                if attempt == self.retry_count - 1:
                    self.logger.error(f"Failed to process batch after {self.retry_count} attempts: {str(e)}")
                    # On final failure, try processing items individually
                    self.logger.info("Falling back to individual item processing")
                    for item in properties:
                        try:
                            self.process_single_item(item, spider)
                        except Exception as e:
                            self.logger.error(f"Failed to process individual item: {str(e)}")
                    return batch_item
                
                self.logger.warning(f"Batch processing attempt {attempt + 1} failed: {str(e)}")
                time.sleep(self.retry_delay)

    def process_single_item(self, item, spider):
        """Process a single property item."""
        for attempt in range(self.retry_count):
            try:
                response = self.session.post(
                    f"{self.api_url}/api/properties",
                    json={
                        'property': dict(item),
                        'spider': spider.name if spider else 'unknown',
                        'city': spider.place if spider else 'unknown'
                    },
                    timeout=30
                )
                response.raise_for_status()
                return item

            except requests.exceptions.RequestException as e:
                if attempt == self.retry_count - 1:
                    self.logger.error(f"Failed to process item after {self.retry_count} attempts: {str(e)}")
                    raise DropItem(f"Failed to process item: {str(e)}")
                
                self.logger.warning(f"Item processing attempt {attempt + 1} failed: {str(e)}")
                time.sleep(self.retry_delay) 
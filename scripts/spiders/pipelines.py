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
        self.api_url = os.getenv('API_URL', 'http://localhost:8080')
        self.session = requests.Session()
        self.logger = logging.getLogger(__name__)
        self.retry_count = 3
        self.retry_delay = 5  # seconds

    def process_item(self, item, spider):
        if item.get('type') == 'properties_batch':
            return self.process_batch(item, spider)
        return self.process_single_item(item, spider)

    def process_batch(self, batch_item, spider):
        """Process a batch of properties."""
        properties = batch_item['items']
        if not properties:
            return batch_item

        for attempt in range(self.retry_count):
            try:
                response = self.session.post(
                    f"{self.api_url}/api/v1/properties/batch",
                    json={
                        'properties': properties,
                        'spider': spider.name,
                        'city': spider.city
                    },
                    timeout=30
                )
                response.raise_for_status()
                self.logger.info(f"Successfully processed batch of {len(properties)} properties")
                return batch_item

            except requests.exceptions.RequestException as e:
                if attempt == self.retry_count - 1:
                    self.logger.error(f"Failed to process batch after {self.retry_count} attempts: {str(e)}")
                    raise DropItem(f"Failed to process batch: {str(e)}")
                
                self.logger.warning(f"Batch processing attempt {attempt + 1} failed: {str(e)}")
                time.sleep(self.retry_delay)

    def process_single_item(self, item, spider):
        """Process a single property item (legacy support)."""
        for attempt in range(self.retry_count):
            try:
                response = self.session.post(
                    f"{self.api_url}/api/v1/properties",
                    json=item,
                    timeout=10
                )
                response.raise_for_status()
                return item

            except requests.exceptions.RequestException as e:
                if attempt == self.retry_count - 1:
                    self.logger.error(f"Failed to process item after {self.retry_count} attempts: {str(e)}")
                    raise DropItem(f"Failed to process item: {str(e)}")
                
                self.logger.warning(f"Processing attempt {attempt + 1} failed: {str(e)}")
                time.sleep(self.retry_delay) 
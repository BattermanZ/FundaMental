import json
from datetime import datetime
import os

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
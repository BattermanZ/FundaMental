#!/usr/bin/env python3
import sys
import os
import json
import logging

# Add the server directory to the Python path
server_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, server_dir)

from scrapy.crawler import CrawlerProcess
from scrapy.utils.project import get_project_settings
from scrapers.funda.spiders.funda_spider import FundaSpider
from scrapers.funda.spiders.funda_spider_sold import FundaSpiderSold
from scrapy import signals
from scrapy.signalmanager import dispatcher
from typing import List, Dict, Any

# Set up logging
logging.basicConfig(
    level=logging.INFO,
    format='{"level":"%(levelname)s","msg":"%(message)s","time":"%(asctime)s"}',
    datefmt='%Y-%m-%dT%H:%M:%S%z'
)

# Configure Scrapy logging to use our format
from scrapy.utils.log import configure_logging
configure_logging(install_root_handler=False)

# Create our custom handler
formatter = logging.Formatter(
    '{"level":"%(levelname)s","msg":"%(message)s","time":"%(asctime)s"}',
    datefmt='%Y-%m-%dT%H:%M:%S%z'
)
handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(formatter)

# Add handler to both our logger and Scrapy's logger
logger = logging.getLogger(__name__)
logger.addHandler(handler)
scrapy_logger = logging.getLogger('scrapy')
scrapy_logger.addHandler(handler)
scrapy_logger.setLevel(logging.INFO)

# Also capture twisted logs
twisted_logger = logging.getLogger('twisted')
twisted_logger.addHandler(handler)
twisted_logger.setLevel(logging.INFO)

class ItemCollector:
    """Collects items and sends them in batches to Go."""
    def __init__(self, batch_size=50):
        self.items: List[Dict[str, Any]] = []
        self.batch_size = batch_size
        self.total_items = 0

    def process_item(self, item):
        """Process a single item."""
        self.items.append(item.to_dict())
        self.total_items += 1
        
        # When we reach batch size, print items as JSON
        if len(self.items) >= self.batch_size:
            self.flush_items()

    def flush_items(self):
        """Flush current items to stdout as JSON."""
        if self.items:
            print(json.dumps({
                "type": "items",
                "data": self.items
            }))
            sys.stdout.flush()
            self.items = []

def run_spider(spider_type, place='amsterdam', max_pages=None, resume=False):
    """
    Run the specified spider with given parameters.
    
    Args:
        spider_type: Either 'active' or 'sold'
        place: City to scrape
        max_pages: Maximum number of pages to scrape
        resume: Whether to resume from previous state (sold spider only)
    """
    try:
        # Initialize settings
        settings = get_project_settings()
        settings.setmodule('scrapers.funda.settings')
        
        # Create item collector
        collector = ItemCollector()
        
        # Set up item collection
        def handle_item(item, response, spider):
            collector.process_item(item)
            return item
        
        # Create crawler process
        process = CrawlerProcess(settings)
        
        # Connect to item signals
        dispatcher.connect(handle_item, signal=signals.item_scraped)
        
        # Run appropriate spider
        if spider_type == 'active':
            process.crawl(FundaSpider, 
                        place=place,
                        max_pages=max_pages)
        elif spider_type == 'sold':
            process.crawl(FundaSpiderSold, 
                        place=place,
                        max_pages=max_pages,
                        resume=resume)
        else:
            raise ValueError(f"Invalid spider type: {spider_type}")
        
        # Start the crawling process
        process.start()
        
        # Flush any remaining items
        collector.flush_items()
        
        return True
        
    except Exception as e:
        logger.error(f"Error running spider: {e}")
        return False

if __name__ == '__main__':
    # Parse command line arguments from Go
    input_data = json.load(sys.stdin)
    
    spider_type = input_data.get('spider_type', 'active')
    place = input_data.get('place', 'amsterdam')
    max_pages = input_data.get('max_pages')
    resume = input_data.get('resume', False)
    
    run_spider(spider_type, place, max_pages, resume) 
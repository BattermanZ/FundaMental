#!/usr/bin/env python3
import sys
import json
import logging
from scrapy.crawler import CrawlerProcess
from scrapy.utils.project import get_project_settings
from scrapers.funda.spiders.funda_spider import FundaSpider
from scrapers.funda.spiders.funda_spider_sold import FundaSpiderSold
from scrapy import signals
from scrapy.signalmanager import dispatcher
from typing import List, Dict, Any

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

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
        
        def handle_spider_closed(spider, reason):
            # Flush any remaining items
            collector.flush_items()
            # Send final statistics
            print(json.dumps({
                "type": "complete",
                "data": {
                    "status": "success",
                    "total_items": collector.total_items,
                    "message": f"Completed scraping {spider_type} listings for {place}"
                }
            }))
            sys.stdout.flush()
        
        # Create crawler process
        process = CrawlerProcess(settings)
        
        # Connect signals
        dispatcher.connect(handle_item, signal=signals.item_scraped)
        dispatcher.connect(handle_spider_closed, signal=signals.spider_closed)
        
        # Choose spider based on type
        if spider_type == 'active':
            spider_class = FundaSpider
            spider_kwargs = {'place': place, 'max_pages': max_pages}
        elif spider_type == 'sold':
            spider_class = FundaSpiderSold
            spider_kwargs = {'place': place, 'max_pages': max_pages, 'resume': resume}
        else:
            raise ValueError(f"Invalid spider type: {spider_type}")
        
        # Configure spider
        process.crawl(spider_class, **spider_kwargs)
        
        # Start crawling
        process.start()
        
    except Exception as e:
        logger.error(f"Error running spider: {e}")
        print(json.dumps({
            "type": "error",
            "data": {
                "status": "error",
                "message": str(e)
            }
        }))
        sys.stdout.flush()
        sys.exit(1)

if __name__ == '__main__':
    # Parse command line arguments from Go
    input_data = json.load(sys.stdin)
    
    spider_type = input_data.get('spider_type', 'active')
    place = input_data.get('place', 'amsterdam')
    max_pages = input_data.get('max_pages')
    resume = input_data.get('resume', False)
    
    run_spider(spider_type, place, max_pages, resume) 
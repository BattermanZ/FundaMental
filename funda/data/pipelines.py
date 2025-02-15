# -*- coding: utf-8 -*-

from funda.data.database import FundaDB

# Define your item pipelines here
#
# Don't forget to add your pipeline to the ITEM_PIPELINES setting
# See: http://doc.scrapy.org/en/latest/topics/item-pipeline.html


class FundaPipeline(object):
    def __init__(self):
        self.db = FundaDB()

    def process_item(self, item, spider):
        # Clean up price (remove currency symbol and convert to int)
        if 'price' in item and item['price']:
            try:
                item['price'] = int(item['price'])
            except ValueError:
                spider.logger.warning(f"Could not convert price to integer: {item['price']}")
                item['price'] = None
        
        # Convert area to integer
        if 'area' in item and item['area']:
            try:
                item['area'] = int(item['area'])
            except ValueError:
                spider.logger.warning(f"Could not convert area to integer: {item['area']}")
                item['area'] = None
        
        # Convert rooms and bedrooms to integer
        for field in ['rooms', 'bedrooms']:
            if field in item and item[field]:
                try:
                    item[field] = int(item[field])
                except ValueError:
                    spider.logger.warning(f"Could not convert {field} to integer: {item[field]}")
                    item[field] = None
        
        # Convert year_built to integer
        if 'year_built' in item and item['year_built']:
            try:
                item['year_built'] = int(item['year_built'])
            except ValueError:
                spider.logger.warning(f"Could not convert year_built to integer: {item['year_built']}")
                item['year_built'] = None

        # Store in database
        self.db.insert_property(item)
        
        return item

    def close_spider(self, spider):
        """When spider finishes, print some basic stats."""
        stats = self.db.get_basic_stats()
        spider.logger.info("Scraping completed! Basic statistics:")
        spider.logger.info(f"Total properties: {stats['total_properties']}")
        spider.logger.info(f"Average price: €{stats['avg_price']:,.2f}")
        spider.logger.info(f"Average days to sell: {stats['avg_days_to_sell']:.1f} days")

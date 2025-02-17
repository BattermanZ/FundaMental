# -*- coding: utf-8 -*-

import re
import json
import sys
from dataclasses import asdict

class FundaPipeline:
    def process_item(self, item, spider):
        try:
            # Clean up price
            if item.price is not None and isinstance(item.price, str):
                try:
                    price_str = item.price.replace('€', '').replace('.', '').replace(',', '').strip()
                    item.price = int(float(price_str))
                except ValueError:
                    spider.logger.warning(f"Could not convert price to integer: {item.price}")
                    item.price = None

            # Convert living_area
            if item.living_area is not None and isinstance(item.living_area, str):
                try:
                    area_str = item.living_area.replace('m²', '').strip()
                    item.living_area = int(float(area_str))
                except ValueError:
                    spider.logger.warning(f"Could not convert living_area to integer: {item.living_area}")
                    item.living_area = None

            # Convert num_rooms
            if item.num_rooms is not None and isinstance(item.num_rooms, str):
                try:
                    rooms_match = re.search(r'(\d+)\s*(?:kamers?|rooms?)', item.num_rooms)
                    if rooms_match:
                        item.num_rooms = int(rooms_match.group(1))
                    else:
                        item.num_rooms = None
                except ValueError:
                    spider.logger.warning(f"Could not convert num_rooms to integer: {item.num_rooms}")
                    item.num_rooms = None

            # Convert year_built
            if item.year_built is not None and isinstance(item.year_built, str):
                try:
                    item.year_built = int(item.year_built.strip())
                except ValueError:
                    spider.logger.warning(f"Could not convert year_built to integer: {item.year_built}")
                    item.year_built = None

        except AttributeError as e:
            spider.logger.error(f"AttributeError processing item: {e}")
            
        return item

class JsonMessagePipeline:
    """Pipeline to format items as JSON messages for the spider manager."""
    
    def process_item(self, item, spider):
        # Convert item to dictionary using to_dict method
        item_dict = item.to_dict()
        
        # Create message
        message = {
            'type': 'items',
            'data': [item_dict]  # Wrap in list since manager expects array
        }
        
        # Write message to stdout
        print(json.dumps(message), flush=True)
        return item
    
    def close_spider(self, spider):
        # Send completion message
        message = {
            'type': 'complete',
            'data': {
                'status': 'success',
                'message': 'Spider completed successfully',
                'total_items': spider.total_items_scraped
            }
        }
        print(json.dumps(message), flush=True) 
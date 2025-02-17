# -*- coding: utf-8 -*-

class FundaPipeline(object):
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

        return item 
# -*- coding: utf-8 -*-

import scrapy

class FundaItem(scrapy.Item):
    url = scrapy.Field()
    street = scrapy.Field()
    city = scrapy.Field()
    postal_code = scrapy.Field()
    price = scrapy.Field()
    year_built = scrapy.Field()
    living_area = scrapy.Field()  # Size in square meters
    num_rooms = scrapy.Field()
    status = scrapy.Field()
    scraped_at = scrapy.Field()

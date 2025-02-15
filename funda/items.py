# -*- coding: utf-8 -*-

import scrapy

class FundaItem(scrapy.Item):
    url = scrapy.Field()
    address = scrapy.Field()
    city = scrapy.Field()
    postal_code = scrapy.Field()
    price = scrapy.Field()
    property_type = scrapy.Field()
    year_built = scrapy.Field()
    rooms = scrapy.Field()
    bedrooms = scrapy.Field()
    area = scrapy.Field()
    title = scrapy.Field()              # Listing title ("Titel")
    posting_date = scrapy.Field()
    sale_date = scrapy.Field()

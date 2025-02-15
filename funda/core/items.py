# -*- coding: utf-8 -*-

import scrapy

class FundaItem(scrapy.Item):
    url = scrapy.Field()
    street = scrapy.Field()
    neighborhood = scrapy.Field()  # Added neighborhood field
    property_type = scrapy.Field()  # Added property type field
    city = scrapy.Field()
    postal_code = scrapy.Field()
    price = scrapy.Field()
    year_built = scrapy.Field()
    living_area = scrapy.Field()  # Size in square meters
    num_rooms = scrapy.Field()
    status = scrapy.Field()
    listing_date = scrapy.Field()  # Date when property was listed
    selling_date = scrapy.Field()  # Date when property was sold
    scraped_at = scrapy.Field()

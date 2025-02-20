import scrapy

class FundaItem(scrapy.Item):
    url = scrapy.Field()
    street = scrapy.Field()
    postal_code = scrapy.Field()
    city = scrapy.Field()
    neighborhood = scrapy.Field()
    price = scrapy.Field()
    year_built = scrapy.Field()
    living_area = scrapy.Field()
    num_rooms = scrapy.Field()
    property_type = scrapy.Field()
    status = scrapy.Field()
    listing_date = scrapy.Field()
    selling_date = scrapy.Field()
    energy_label = scrapy.Field()
    scraped_at = scrapy.Field() 
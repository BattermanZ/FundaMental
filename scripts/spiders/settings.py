BOT_NAME = 'funda_test'

SPIDER_MODULES = ['spiders']
NEWSPIDER_MODULE = 'spiders'

# Crawl responsibly by identifying yourself
USER_AGENT = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36'

# Configure maximum concurrent requests
CONCURRENT_REQUESTS = 1
DOWNLOAD_DELAY = 3

# Disable cookies for development
COOKIES_ENABLED = False

# Configure item pipelines
ITEM_PIPELINES = {
   'spiders.pipelines.TestPipeline': 300,
   'spiders.pipelines.JsonExportPipeline': 500,
}

# Enable and configure HTTP caching for development
HTTPCACHE_ENABLED = True
HTTPCACHE_EXPIRATION_SECS = 0
HTTPCACHE_DIR = 'httpcache'
HTTPCACHE_IGNORE_HTTP_CODES = []
HTTPCACHE_STORAGE = 'scrapy.extensions.httpcache.FilesystemCacheStorage'

# Disable redirects for faster testing
REDIRECT_ENABLED = False

# Enable logging for development
LOG_LEVEL = 'DEBUG'
LOG_FORMAT = '%(levelname)s: %(message)s' 
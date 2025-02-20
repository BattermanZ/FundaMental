BOT_NAME = 'funda_test'

SPIDER_MODULES = ['spiders']
NEWSPIDER_MODULE = 'spiders'

# Crawl responsibly by identifying yourself
USER_AGENT = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36'

# Configure request settings
DOWNLOAD_DELAY = 2
CONCURRENT_REQUESTS = 2
CONCURRENT_REQUESTS_PER_DOMAIN = 2

# Configure auto-throttling
AUTOTHROTTLE_ENABLED = True
AUTOTHROTTLE_START_DELAY = 2
AUTOTHROTTLE_MAX_DELAY = 30
AUTOTHROTTLE_TARGET_CONCURRENCY = 2.0
DOWNLOAD_TIMEOUT = 30

# Disable cookies for development
COOKIES_ENABLED = False

# Configure item pipelines
ITEM_PIPELINES = {
    'spiders.pipelines.FundaPipeline': 300,
}

# Batch processing settings
BATCH_SIZE = 100

# Retry settings
RETRY_ENABLED = True
RETRY_TIMES = 3
RETRY_HTTP_CODES = [500, 502, 503, 504, 408, 429]
RETRY_PRIORITY_ADJUST = -1

# Enable and configure logging
LOG_ENABLED = True
LOG_LEVEL = 'INFO'
LOG_FORMAT = '%(asctime)s [%(name)s] %(levelname)s: %(message)s'

# Enable memory monitoring
MEMUSAGE_ENABLED = True
MEMUSAGE_WARNING_MB = 512
MEMUSAGE_LIMIT_MB = 1024
MEMUSAGE_CHECK_INTERVAL_SECONDS = 60

# Enable and configure HTTP caching for development
HTTPCACHE_ENABLED = True
HTTPCACHE_EXPIRATION_SECS = 0
HTTPCACHE_DIR = 'httpcache'
HTTPCACHE_IGNORE_HTTP_CODES = []
HTTPCACHE_STORAGE = 'scrapy.extensions.httpcache.FilesystemCacheStorage'

# Disable redirects for faster testing
REDIRECT_ENABLED = False

# Add randomization to appear more human-like
RANDOMIZE_DOWNLOAD_DELAY = True
DOWNLOAD_DELAY_RANDOMIZATION_FACTOR = 0.5  # Will vary delay between 1.5 and 4.5 seconds 
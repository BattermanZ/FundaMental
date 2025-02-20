package config

import "github.com/caarlos0/env/v6"

type Config struct {
    // BatchProcessing configuration
    BatchProcessing struct {
        // Maximum number of properties to accumulate before processing
        MaxBatchSize int `env:"BATCH_MAX_SIZE" envDefault:"100"`
        
        // Maximum time to wait before processing a non-full batch (in seconds)
        MaxBatchWaitTime int `env:"BATCH_WAIT_TIME" envDefault:"30"`
        
        // Number of concurrent batch processors
        ProcessorCount int `env:"BATCH_PROCESSOR_COUNT" envDefault:"2"`
        
        // Maximum number of retries for failed batches
        MaxRetries int `env:"BATCH_MAX_RETRIES" envDefault:"3"`
        
        // Delay between retries in seconds
        RetryDelay int `env:"BATCH_RETRY_DELAY" envDefault:"5"`
    }
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}
package processor

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"fundamental/server/config"
	"fundamental/server/internal/database"
	"fundamental/server/internal/models"
	"fundamental/server/internal/queue"
)

func generateTestProperties(count int) []*models.Property {
	properties := make([]*models.Property, count)
	for i := range properties {
		properties[i] = &models.Property{
			Address:    fmt.Sprintf("Test Address %d", i),
			Price:      500000 + (i * 1000),
			City:       "Amsterdam",
			PostalCode: fmt.Sprintf("1000%d", i),
		}
	}
	return properties
}

func BenchmarkBatchProcessing(b *testing.B) {
	// Setup test database
	db, err := database.NewTestDB()
	require.NoError(b, err)
	err = database.MigrateSchema(db)
	require.NoError(b, err)

	// Test configurations
	batchSizes := []int{10, 50, 100, 500}
	propertyCounts := []int{1000, 5000, 10000}

	for _, batchSize := range batchSizes {
		for _, propertyCount := range propertyCounts {
			b.Run(fmt.Sprintf("BatchSize_%d_Properties_%d", batchSize, propertyCount), func(b *testing.B) {
				// Setup configuration
				cfg := &config.Config{}
				cfg.BatchProcessing.ProcessorCount = 4
				cfg.BatchProcessing.MaxRetries = 3
				cfg.BatchProcessing.MaxBatchSize = batchSize
				logger := logrus.New()
				logger.SetLevel(logrus.WarnLevel) // Reduce logging noise during benchmarks

				// Create components
				propertyQueue := queue.NewPropertyQueue(batchSize)
				processor := NewBatchProcessor(db, propertyQueue, cfg, logger)

				// Generate test data
				properties := generateTestProperties(propertyCount)

				// Start processor
				processor.Start()
				defer processor.Stop()

				// Reset timer before the actual benchmark
				b.ResetTimer()

				// Run benchmark
				for i := 0; i < b.N; i++ {
					// Clear database before each iteration
					b.StopTimer()
					db.Exec("DELETE FROM properties")
					b.StartTimer()

					// Push properties to queue
					startTime := time.Now()
					for _, prop := range properties {
						err := propertyQueue.Push(prop)
						require.NoError(b, err)
					}

					// Wait for processing to complete
					time.Sleep(time.Duration(float64(propertyCount) * 0.1 * float64(time.Millisecond)))

					// Record metrics
					duration := time.Since(startTime)
					throughput := float64(propertyCount) / duration.Seconds()
					b.ReportMetric(throughput, "properties/sec")

					// Verify all properties were processed
					var count int64
					result := db.Model(&models.Property{}).Count(&count)
					require.NoError(b, result.Error)
					require.Equal(b, int64(propertyCount), count)
				}
			})
		}
	}
}

func BenchmarkConcurrentBatchProcessing(b *testing.B) {
	// Setup test database
	db, err := database.NewTestDB()
	require.NoError(b, err)
	err = database.MigrateSchema(db)
	require.NoError(b, err)

	// Test configurations
	concurrencyLevels := []int{2, 4, 8}
	propertyCount := 10000
	batchSize := 100

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			// Setup configuration
			cfg := &config.Config{}
			cfg.BatchProcessing.ProcessorCount = concurrency
			cfg.BatchProcessing.MaxRetries = 3
			cfg.BatchProcessing.MaxBatchSize = batchSize
			logger := logrus.New()
			logger.SetLevel(logrus.WarnLevel)

			// Create components
			propertyQueue := queue.NewPropertyQueue(batchSize)
			processor := NewBatchProcessor(db, propertyQueue, cfg, logger)

			// Generate test data
			properties := generateTestProperties(propertyCount)

			// Start processor
			processor.Start()
			defer processor.Stop()

			// Reset timer before the actual benchmark
			b.ResetTimer()

			// Run benchmark
			for i := 0; i < b.N; i++ {
				// Clear database before each iteration
				b.StopTimer()
				db.Exec("DELETE FROM properties")
				b.StartTimer()

				// Push properties to queue using multiple goroutines
				startTime := time.Now()
				batchesPerWorker := propertyCount / (batchSize * concurrency)

				var wg sync.WaitGroup
				for w := 0; w < concurrency; w++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						start := workerID * batchesPerWorker * batchSize
						end := start + batchesPerWorker*batchSize
						for j := start; j < end; j++ {
							err := propertyQueue.Push(properties[j])
							require.NoError(b, err)
						}
					}(w)
				}
				wg.Wait()

				// Wait for processing to complete
				time.Sleep(time.Duration(float64(propertyCount) * 0.1 * float64(time.Millisecond)))

				// Record metrics
				duration := time.Since(startTime)
				throughput := float64(propertyCount) / duration.Seconds()
				b.ReportMetric(throughput, "properties/sec")

				// Verify all properties were processed
				var count int64
				result := db.Model(&models.Property{}).Count(&count)
				require.NoError(b, result.Error)
				require.Equal(b, int64(propertyCount), count)
			}
		})
	}
}

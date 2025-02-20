package processor

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"fundamental/server/config"
	"fundamental/server/internal/database"
	"fundamental/server/internal/models"
	"fundamental/server/internal/queue"
)

func getMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

func TestMemoryUsageWithDifferentBatchSizes(t *testing.T) {
	// Setup test database
	db, err := database.NewTestDB()
	require.NoError(t, err)
	err = database.MigrateSchema(db)
	require.NoError(t, err)

	// Test configurations
	batchSizes := []int{10, 50, 100, 500, 1000}
	propertyCount := 10000
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	for _, batchSize := range batchSizes {
		t.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(t *testing.T) {
			// Force GC before test
			runtime.GC()
			baseMemStats := getMemStats()

			// Setup configuration
			cfg := &config.Config{}
			cfg.BatchProcessing.ProcessorCount = 4
			cfg.BatchProcessing.MaxRetries = 3
			cfg.BatchProcessing.MaxBatchSize = batchSize

			// Create components
			propertyQueue := queue.NewPropertyQueue(batchSize)
			processor := NewBatchProcessor(db, propertyQueue, cfg, logger)

			// Generate test data
			properties := generateTestProperties(propertyCount)

			// Start processor
			processor.Start()
			defer processor.Stop()

			// Record memory before processing
			beforeMemStats := getMemStats()

			// Process properties
			startTime := time.Now()
			for _, prop := range properties {
				err := propertyQueue.Push(prop)
				require.NoError(t, err)
			}

			// Wait for processing to complete
			time.Sleep(time.Duration(float64(propertyCount) * 0.1 * float64(time.Millisecond)))

			// Record memory after processing
			afterMemStats := getMemStats()

			// Calculate memory usage
			memoryIncrease := afterMemStats.Alloc - beforeMemStats.Alloc
			peakMemory := afterMemStats.TotalAlloc - baseMemStats.TotalAlloc
			duration := time.Since(startTime)

			// Log memory metrics
			t.Logf("Batch Size: %d", batchSize)
			t.Logf("Memory Increase: %.2f MB", float64(memoryIncrease)/1024/1024)
			t.Logf("Peak Memory Usage: %.2f MB", float64(peakMemory)/1024/1024)
			t.Logf("Processing Duration: %v", duration)
			t.Logf("Properties/Second: %.2f", float64(propertyCount)/duration.Seconds())

			// Verify processing completed
			var count int64
			result := db.Model(&models.Property{}).Count(&count)
			require.NoError(t, result.Error)
			require.Equal(t, int64(propertyCount), count)

			// Memory usage assertions
			maxAllowedMemoryPerProperty := float64(1024) // 1KB per property
			memoryPerProperty := float64(memoryIncrease) / float64(propertyCount)
			require.Less(t, memoryPerProperty, maxAllowedMemoryPerProperty,
				"Memory usage per property exceeded limit")
		})
	}
}

func TestMemoryLeakCheck(t *testing.T) {
	// Setup test database
	db, err := database.NewTestDB()
	require.NoError(t, err)
	err = database.MigrateSchema(db)
	require.NoError(t, err)

	// Configuration
	cfg := &config.Config{}
	cfg.BatchProcessing.ProcessorCount = 4
	cfg.BatchProcessing.MaxRetries = 3
	cfg.BatchProcessing.MaxBatchSize = 100
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Create components
	propertyQueue := queue.NewPropertyQueue(cfg.BatchProcessing.MaxBatchSize)
	processor := NewBatchProcessor(db, propertyQueue, cfg, logger)

	// Start processor
	processor.Start()
	defer processor.Stop()

	// Run multiple processing cycles
	cycles := 5
	propertiesPerCycle := 1000
	baseMemStats := getMemStats()
	var lastCycleMemory uint64

	for cycle := 0; cycle < cycles; cycle++ {
		// Force GC before cycle
		runtime.GC()
		beforeCycleStats := getMemStats()

		// Process batch
		properties := generateTestProperties(propertiesPerCycle)
		for _, prop := range properties {
			err := propertyQueue.Push(prop)
			require.NoError(t, err)
		}

		// Wait for processing
		time.Sleep(time.Duration(float64(propertiesPerCycle) * 0.1 * float64(time.Millisecond)))

		// Clear database
		db.Exec("DELETE FROM properties")

		// Record memory after cycle
		afterCycleStats := getMemStats()
		cycleMemory := afterCycleStats.Alloc - beforeCycleStats.Alloc

		// Log cycle metrics
		t.Logf("Cycle %d Memory Usage: %.2f MB", cycle+1, float64(cycleMemory)/1024/1024)

		if cycle > 0 {
			// Check for memory growth
			memoryIncrease := float64(cycleMemory) - float64(lastCycleMemory)
			maxAllowedIncrease := float64(1024 * 1024) // 1MB
			require.Less(t, memoryIncrease, maxAllowedIncrease,
				"Detected potential memory leak: significant memory growth between cycles")
		}

		lastCycleMemory = cycleMemory
	}

	// Final memory check
	finalMemStats := getMemStats()
	totalMemoryGrowth := finalMemStats.Alloc - baseMemStats.Alloc
	t.Logf("Total Memory Growth: %.2f MB", float64(totalMemoryGrowth)/1024/1024)
}

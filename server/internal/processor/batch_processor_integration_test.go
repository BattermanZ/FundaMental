package processor

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"fundamental/server/config"
	"fundamental/server/internal/database"
	"fundamental/server/internal/models"
	"fundamental/server/internal/queue"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Setup test database connection
	db, err := database.NewTestDB()
	require.NoError(t, err)

	// Migrate the schema
	err = database.MigrateSchema(db)
	require.NoError(t, err)

	return db
}

func TestBatchProcessingIntegration(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	cfg := &config.Config{}
	cfg.BatchProcessing.ProcessorCount = 2
	cfg.BatchProcessing.MaxRetries = 3
	cfg.BatchProcessing.MaxBatchSize = 100
	logger := logrus.New()

	// Create components
	propertyQueue := queue.NewPropertyQueue(cfg.BatchProcessing.MaxBatchSize)
	processor := NewBatchProcessor(db, propertyQueue, cfg, logger)

	// Start processor
	processor.Start()
	defer processor.Stop()

	// Create test data
	testProperties := []*models.Property{
		{
			Address:    "Test Address 1",
			Price:      500000,
			City:       "Amsterdam",
			PostalCode: "1000AA",
		},
		{
			Address:    "Test Address 2",
			Price:      600000,
			City:       "Amsterdam",
			PostalCode: "1000BB",
		},
	}

	// Push properties to queue
	for _, prop := range testProperties {
		err := propertyQueue.Push(prop)
		require.NoError(t, err)
	}

	// Allow time for processing
	time.Sleep(2 * time.Second)

	// Verify properties were stored
	for _, expectedProp := range testProperties {
		var storedProp models.Property
		result := db.Where("postal_code = ?", expectedProp.PostalCode).First(&storedProp)
		assert.NoError(t, result.Error)
		assert.Equal(t, expectedProp.Address, storedProp.Address)
		assert.Equal(t, expectedProp.Price, storedProp.Price)
		assert.Equal(t, expectedProp.City, storedProp.City)
	}
}

func TestBatchProcessingWithConcurrency(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	cfg := &config.Config{}
	cfg.BatchProcessing.ProcessorCount = 4
	cfg.BatchProcessing.MaxRetries = 3
	cfg.BatchProcessing.MaxBatchSize = 50
	logger := logrus.New()

	// Create components
	propertyQueue := queue.NewPropertyQueue(cfg.BatchProcessing.MaxBatchSize)
	processor := NewBatchProcessor(db, propertyQueue, cfg, logger)

	// Start processor
	processor.Start()
	defer processor.Stop()

	// Create large test dataset
	testBatches := make([][]*models.Property, 5)
	for i := range testBatches {
		batch := make([]*models.Property, 20)
		for j := range batch {
			batch[j] = &models.Property{
				Address:    fmt.Sprintf("Test Address %d-%d", i, j),
				Price:      500000 + (i * 100000) + (j * 1000),
				City:       "Amsterdam",
				PostalCode: fmt.Sprintf("1000%d%d", i, j),
			}
		}
		testBatches[i] = batch
	}

	// Push properties concurrently
	var wg sync.WaitGroup
	for _, batch := range testBatches {
		wg.Add(1)
		go func(props []*models.Property) {
			defer wg.Done()
			for _, prop := range props {
				err := propertyQueue.Push(prop)
				require.NoError(t, err)
			}
		}(batch)
	}

	// Wait for all pushes to complete
	wg.Wait()

	// Allow time for processing
	time.Sleep(5 * time.Second)

	// Verify all properties were stored
	var count int64
	result := db.Model(&models.Property{}).Count(&count)
	assert.NoError(t, result.Error)
	assert.Equal(t, int64(100), count) // 5 batches * 20 properties
}

func TestBatchProcessingErrorRecovery(t *testing.T) {
	// Setup with mock DB that fails initially
	mockDB := &MockDB{}
	cfg := &config.Config{}
	cfg.BatchProcessing.ProcessorCount = 2
	cfg.BatchProcessing.MaxRetries = 3
	cfg.BatchProcessing.RetryDelay = 1
	logger := logrus.New()

	propertyQueue := queue.NewPropertyQueue(10)
	processor := NewBatchProcessor(mockDB, propertyQueue, cfg, logger)

	// Configure mock to fail twice then succeed
	var attemptCount int
	mockDB.On("Transaction", mock.Anything).Return(func(fc func(*gorm.DB) error) error {
		attemptCount++
		if attemptCount <= 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	// Start processor
	processor.Start()
	defer processor.Stop()

	// Push test property
	testProp := &models.Property{
		Address:    "Test Address",
		Price:      500000,
		City:       "Amsterdam",
		PostalCode: "1000AA",
	}
	err := propertyQueue.Push(testProp)
	require.NoError(t, err)

	// Allow time for processing and retries
	time.Sleep(5 * time.Second)

	// Verify the number of attempts
	assert.Equal(t, 3, attemptCount)
	mockDB.AssertExpectations(t)
}

package processor

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"fundamental/server/config"
	"fundamental/server/internal/models"
	"fundamental/server/internal/queue"
)

// MockDB is a mock implementation of *gorm.DB
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Transaction(fc func(*gorm.DB) error, opts ...*sql.TxOptions) error {
	args := m.Called(fc)
	return args.Error(0)
}

func TestNewBatchProcessor(t *testing.T) {
	// Setup
	mockDB := &MockDB{}
	mockQueue := queue.NewPropertyQueue(10)
	cfg := &config.Config{}
	cfg.BatchProcessing.ProcessorCount = 2
	cfg.BatchProcessing.MaxRetries = 3
	logger := logrus.New()

	// Test
	processor := NewBatchProcessor(mockDB, mockQueue, cfg, logger)

	// Assert
	assert.NotNil(t, processor)
	assert.Equal(t, mockDB, processor.db)
	assert.Equal(t, mockQueue, processor.queue)
	assert.Equal(t, cfg, processor.config)
	assert.Equal(t, logger, processor.logger)
}

func TestBatchProcessor_ProcessBatch(t *testing.T) {
	// Setup
	mockDB := &MockDB{}
	mockQueue := queue.NewPropertyQueue(10)
	cfg := &config.Config{}
	cfg.BatchProcessing.ProcessorCount = 2
	cfg.BatchProcessing.MaxRetries = 3
	cfg.BatchProcessing.RetryDelay = 1
	logger := logrus.New()

	processor := NewBatchProcessor(mockDB, mockQueue, cfg, logger)

	batch := []*models.Property{
		{ID: 1, Address: "Test Address 1"},
		{ID: 2, Address: "Test Address 2"},
	}

	// Test successful processing
	mockDB.On("Transaction", mock.Anything).Return(nil).Once()
	err := processor.processBatch(batch)
	assert.NoError(t, err)

	// Test retry on failure
	mockDB.On("Transaction", mock.Anything).Return(errors.New("db error")).Times(3)
	err = processor.processBatch(batch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process batch after 3 attempts")
}

func TestBatchProcessor_StartStop(t *testing.T) {
	// Setup
	mockDB := &MockDB{}
	mockQueue := queue.NewPropertyQueue(10)
	cfg := &config.Config{}
	cfg.BatchProcessing.ProcessorCount = 2
	logger := logrus.New()

	processor := NewBatchProcessor(mockDB, mockQueue, cfg, logger)

	// Test Start
	processor.Start()
	time.Sleep(100 * time.Millisecond) // Give time for goroutines to start

	// Test Stop
	processor.Stop()
	// Verify graceful shutdown
	mockQueue.Close()
	assert.True(t, mockQueue.IsClosed())
}

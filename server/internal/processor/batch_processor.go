package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"fundamental/server/config"
	"fundamental/server/internal/database"
	"fundamental/server/internal/models"
	"fundamental/server/internal/queue"
)

// BatchProcessor handles the processing of property batches
type BatchProcessor struct {
	db        *gorm.DB
	logger    *logrus.Logger
	config    *config.Config
	queue     *queue.PropertyQueue
	waitGroup sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewBatchProcessor creates a new batch processor instance
func NewBatchProcessor(db *gorm.DB, queue *queue.PropertyQueue, config *config.Config, logger *logrus.Logger) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &BatchProcessor{
		db:     db,
		queue:  queue,
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins processing batches from the queue
func (p *BatchProcessor) Start() {
	for i := 0; i < p.config.BatchProcessing.ProcessorCount; i++ {
		p.waitGroup.Add(1)
		go p.processLoop()
	}
}

// Stop gracefully shuts down the processor
func (p *BatchProcessor) Stop() {
	p.cancel()
	p.waitGroup.Wait()
}

// processLoop handles the continuous processing of batches
func (p *BatchProcessor) processLoop() {
	defer p.waitGroup.Done()

	p.queue.Subscribe(func(batch []*models.Property) error {
		return p.processBatch(batch)
	})
}

// processBatch handles a single batch of properties with transaction and retry logic
func (p *BatchProcessor) processBatch(batch []*models.Property) error {
	var err error
	for attempt := 0; attempt <= p.config.BatchProcessing.MaxRetries; attempt++ {
		if attempt > 0 {
			p.logger.Infof("Retrying batch processing, attempt %d of %d", attempt, p.config.BatchProcessing.MaxRetries)
			time.Sleep(time.Duration(p.config.BatchProcessing.RetryDelay) * time.Second)
		}

		err = p.db.Transaction(func(tx *gorm.DB) error {
			if err := database.UpsertProperties(tx, batch); err != nil {
				return fmt.Errorf("failed to upsert properties batch: %w", err)
			}
			return nil
		})

		if err == nil {
			p.logger.Infof("Successfully processed batch of %d properties", len(batch))
			return nil
		}

		p.logger.Errorf("Batch processing failed: %v", err)
	}

	return fmt.Errorf("failed to process batch after %d attempts: %w", p.config.BatchProcessing.MaxRetries, err)
}

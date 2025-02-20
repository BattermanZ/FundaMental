package queue

import (
	"errors"
	"fundamental/server/internal/models"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	ErrQueueFull   = errors.New("queue is full")
	ErrQueueClosed = errors.New("queue is closed")
)

// PropertyQueue represents an in-memory queue for property batches
type PropertyQueue struct {
	items    chan []*models.Property
	done     chan struct{}
	maxSize  int
	closed   bool
	mu       sync.RWMutex
	logger   *logrus.Logger
	handlers []func([]*models.Property) error
}

// NewPropertyQueue creates a new property queue with the specified buffer size
func NewPropertyQueue(bufferSize int, logger *logrus.Logger) *PropertyQueue {
	return &PropertyQueue{
		items:    make(chan []*models.Property, bufferSize),
		done:     make(chan struct{}),
		maxSize:  bufferSize,
		logger:   logger,
		handlers: make([]func([]*models.Property) error, 0),
	}
}

// Push adds a batch of properties to the queue
func (q *PropertyQueue) Push(properties []*models.Property) error {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return ErrQueueClosed
	}
	q.mu.RUnlock()

	// Non-blocking send to prevent deadlocks
	select {
	case q.items <- properties:
		q.logger.WithField("batch_size", len(properties)).Debug("Pushed batch to queue")
		return nil
	default:
		return ErrQueueFull
	}
}

// Subscribe adds a handler function that will be called for each batch
func (q *PropertyQueue) Subscribe(handler func([]*models.Property) error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers = append(q.handlers, handler)
}

// Start begins processing items in the queue
func (q *PropertyQueue) Start() {
	go q.process()
}

// process handles the queue processing loop
func (q *PropertyQueue) process() {
	for {
		select {
		case <-q.done:
			return
		case batch := <-q.items:
			q.processBatch(batch)
		}
	}
}

// processBatch sends the batch to all subscribed handlers
func (q *PropertyQueue) processBatch(batch []*models.Property) {
	q.mu.RLock()
	handlers := q.handlers
	q.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(batch); err != nil {
			q.logger.WithError(err).Error("Handler failed to process batch")
		}
	}
}

// Close stops the queue and prevents new items from being added
func (q *PropertyQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.done)
	close(q.items)
	return nil
}

// Len returns the current number of batches in the queue
func (q *PropertyQueue) Len() int {
	return len(q.items)
}

// IsClosed returns whether the queue has been closed
func (q *PropertyQueue) IsClosed() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.closed
}

package queue

import (
    "fundamental/server/internal/models"
    "sync"
    "testing"
    "time"

    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
)

func TestNewPropertyQueue(t *testing.T) {
    logger := logrus.New()
    q := NewPropertyQueue(10, logger)
    assert.NotNil(t, q)
    assert.Equal(t, 10, q.maxSize)
    assert.False(t, q.IsClosed())
}

func TestPropertyQueue_Push(t *testing.T) {
    logger := logrus.New()
    q := NewPropertyQueue(2, logger)

    // Test successful push
    props := []*models.Property{{URL: "test1"}}
    err := q.Push(props)
    assert.NoError(t, err)
    assert.Equal(t, 1, q.Len())

    // Test queue full
    for i := 0; i < 2; i++ {
        props := []*models.Property{{URL: "test"}}
        _ = q.Push(props)
    }
    err = q.Push(props)
    assert.Equal(t, ErrQueueFull, err)

    // Test closed queue
    q.Close()
    err = q.Push(props)
    assert.Equal(t, ErrQueueClosed, err)
}

func TestPropertyQueue_Subscribe(t *testing.T) {
    logger := logrus.New()
    q := NewPropertyQueue(10, logger)

    var processed []*models.Property
    var mu sync.Mutex

    // Add handler
    q.Subscribe(func(props []*models.Property) error {
        mu.Lock()
        processed = append(processed, props...)
        mu.Unlock()
        return nil
    })

    // Start queue
    q.Start()

    // Push items
    testProps := []*models.Property{{URL: "test1"}, {URL: "test2"}}
    err := q.Push(testProps)
    assert.NoError(t, err)

    // Wait for processing
    time.Sleep(100 * time.Millisecond)

    // Verify processing
    mu.Lock()
    assert.Equal(t, 2, len(processed))
    assert.Equal(t, "test1", processed[0].URL)
    assert.Equal(t, "test2", processed[1].URL)
    mu.Unlock()
}

func TestPropertyQueue_Close(t *testing.T) {
    logger := logrus.New()
    q := NewPropertyQueue(10, logger)

    // Test first close
    err := q.Close()
    assert.NoError(t, err)
    assert.True(t, q.IsClosed())

    // Test second close (should be no-op)
    err = q.Close()
    assert.NoError(t, err)
}

func TestPropertyQueue_ProcessBatch(t *testing.T) {
    logger := logrus.New()
    q := NewPropertyQueue(10, logger)

    var wg sync.WaitGroup
    processedBatches := 0
    var mu sync.Mutex

    // Add multiple handlers
    for i := 0; i < 3; i++ {
        wg.Add(1)
        q.Subscribe(func(props []*models.Property) error {
            mu.Lock()
            processedBatches++
            mu.Unlock()
            wg.Done()
            return nil
        })
    }

    // Start queue
    q.Start()

    // Push a batch
    testProps := []*models.Property{{URL: "test"}}
    err := q.Push(testProps)
    assert.NoError(t, err)

    // Wait for all handlers
    wg.Wait()

    // Verify all handlers processed the batch
    mu.Lock()
    assert.Equal(t, 3, processedBatches)
    mu.Unlock()
} 
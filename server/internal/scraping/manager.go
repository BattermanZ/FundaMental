package scraping

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"fundamental/server/internal/database"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// SpiderManager handles the execution of Scrapy spiders
type SpiderManager struct {
	logger     *logrus.Logger
	scriptPath string
	db         *database.Database
}

// SpiderParams contains parameters for running a spider
type SpiderParams struct {
	SpiderType string `json:"spider_type"` // "active" or "sold"
	Place      string `json:"place"`       // e.g., "amsterdam"
	MaxPages   *int   `json:"max_pages"`   // optional max pages to scrape
	Resume     bool   `json:"resume"`      // whether to resume from previous state
}

// SpiderMessage represents a message from the Python script
type SpiderMessage struct {
	Type string          `json:"type"` // "items", "complete", or "error"
	Data json.RawMessage `json:"data"`
}

// NewSpiderManager creates a new spider manager
func NewSpiderManager(db *database.Database, logger *logrus.Logger) *SpiderManager {
	if logger == nil {
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetOutput(os.Stdout)
	}

	// Get the absolute path to the script
	scriptPath := filepath.Join("scripts", "run_spider.py")
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		logger.WithError(err).Error("Failed to get absolute path to spider script")
	}

	return &SpiderManager{
		logger:     logger,
		scriptPath: absPath,
		db:         db,
	}
}

// RunSpider executes a spider with the given parameters
func (m *SpiderManager) RunSpider(params SpiderParams) error {
	m.logger.WithFields(logrus.Fields{
		"spider_type": params.SpiderType,
		"place":       params.Place,
		"max_pages":   params.MaxPages,
		"resume":      params.Resume,
	}).Info("Starting spider")

	// Convert parameters to JSON
	inputData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal spider parameters: %w", err)
	}

	// Create command to run the Python script
	cmd := exec.Command("python3", m.scriptPath)
	cmd.Stdin = bytes.NewBuffer(inputData)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start spider: %w", err)
	}

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Process stdout in a goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var msg SpiderMessage
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				m.logger.WithError(err).Error("Failed to parse spider message")
				continue
			}

			switch msg.Type {
			case "items":
				// Parse and store items
				var items []map[string]interface{}
				if err := json.Unmarshal(msg.Data, &items); err != nil {
					m.logger.WithError(err).Error("Failed to parse items")
					continue
				}
				if err := m.db.InsertProperties(items); err != nil {
					m.logger.WithError(err).Error("Failed to store items")
				}

			case "complete":
				// Parse completion message
				var complete struct {
					Status     string `json:"status"`
					Message    string `json:"message"`
					TotalItems int    `json:"total_items"`
				}
				if err := json.Unmarshal(msg.Data, &complete); err != nil {
					m.logger.WithError(err).Error("Failed to parse completion message")
					continue
				}
				m.logger.WithFields(logrus.Fields{
					"status":      complete.Status,
					"message":     complete.Message,
					"total_items": complete.TotalItems,
				}).Info("Spider completed")

			case "error":
				// Parse error message
				var errMsg struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal(msg.Data, &errMsg); err != nil {
					m.logger.WithError(err).Error("Failed to parse error message")
					continue
				}
				m.logger.WithField("message", errMsg.Message).Error("Spider error")
			}
		}
		if err := scanner.Err(); err != nil {
			m.logger.WithError(err).Error("Scanner error")
		}
	}()

	// Process stderr in a goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			m.logger.Error(scanner.Text())
		}
	}()

	// Wait for command completion in a goroutine
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for completion or timeout
	if err := <-done; err != nil {
		return fmt.Errorf("spider execution failed: %w", err)
	}

	return nil
}

// RunActiveSpider runs the active listings spider
func (m *SpiderManager) RunActiveSpider(place string, maxPages *int) error {
	return m.RunSpider(SpiderParams{
		SpiderType: "active",
		Place:      place,
		MaxPages:   maxPages,
	})
}

// RunSoldSpider runs the sold listings spider
func (m *SpiderManager) RunSoldSpider(place string, maxPages *int, resume bool) error {
	return m.RunSpider(SpiderParams{
		SpiderType: "sold",
		Place:      place,
		MaxPages:   maxPages,
		Resume:     resume,
	})
}

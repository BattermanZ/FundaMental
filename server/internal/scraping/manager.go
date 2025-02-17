package scraping

import (
	"bufio"
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

	// Prepare the command
	cmd := exec.Command("python3", m.scriptPath)

	// Prepare input data
	input := map[string]interface{}{
		"spider_type": params.SpiderType,
		"place":       params.Place,
		"max_pages":   params.MaxPages,
		"resume":      params.Resume,
	}

	// Convert input to JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input data: %v", err)
	}

	// Create pipes for stdin and stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	// Combine stdout and stderr
	combinedOutput, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	cmd.Stderr = cmd.Stdout

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start spider: %v", err)
	}

	// Write input data
	if _, err := stdin.Write(inputJSON); err != nil {
		return fmt.Errorf("failed to write input data: %v", err)
	}
	stdin.Close()

	// Read output
	scanner := bufio.NewScanner(combinedOutput)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // Increase buffer size to 1MB

	for scanner.Scan() {
		line := scanner.Bytes()

		// First try to parse as a log message
		var logMessage struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
			Time  string `json:"time"`
		}
		if err := json.Unmarshal(line, &logMessage); err == nil {
			// Forward the log message using the appropriate log level
			switch logMessage.Level {
			case "ERROR":
				m.logger.Error(logMessage.Msg)
			case "WARNING":
				m.logger.Warn(logMessage.Msg)
			case "INFO":
				m.logger.Info(logMessage.Msg)
			case "DEBUG":
				m.logger.Debug(logMessage.Msg)
			}
			continue
		}

		// If not a log message, try parsing as a spider message
		var message SpiderMessage
		if err := json.Unmarshal(line, &message); err != nil {
			// If we can't parse it as JSON at all, just log it as debug
			m.logger.Debug(string(line))
			continue
		}

		switch message.Type {
		case "items":
			// Process scraped items
			var items []map[string]interface{}
			if err := json.Unmarshal(message.Data, &items); err != nil {
				m.logger.WithError(err).Error("Failed to parse items data")
				continue
			}
			m.logger.WithField("items", items).Info("Received items from spider")
			// Process items using InsertProperties
			if err := m.db.InsertProperties(items); err != nil {
				m.logger.WithError(err).Error("Failed to store properties")
			}
		case "error":
			var errorData map[string]interface{}
			if err := json.Unmarshal(message.Data, &errorData); err != nil {
				m.logger.WithError(err).Error("Failed to parse error data")
				continue
			}
			m.logger.WithField("error", errorData).Error("Spider error")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading spider output: %v", err)
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("spider failed: %v", err)
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

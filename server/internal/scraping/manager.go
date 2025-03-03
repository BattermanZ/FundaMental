package scraping

import (
	"bufio"
	"encoding/json"
	"fmt"
	"fundamental/server/internal/database"
	"os"
	"os/exec"
	"path/filepath"

	"fundamental/server/internal/geocoding"
	"fundamental/server/internal/telegram"

	"github.com/sirupsen/logrus"
)

// SpiderManager handles the execution of Scrapy spiders
type SpiderManager struct {
	logger          *logrus.Logger
	scriptPath      string
	db              *database.Database
	geocoder        *geocoding.Geocoder
	telegramService *telegram.Service
}

// SpiderParams contains parameters for running a spider
type SpiderParams struct {
	SpiderType string `json:"spider_type"` // "active" or "sold"
	Place      string `json:"place"`       // normalized city name (e.g., "den-bosch" not "'s-Hertogenbosch")
	MaxPages   *int   `json:"max_pages"`   // optional max pages to scrape
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
		logger.SetLevel(logrus.DebugLevel)
	}

	// Get the absolute path to the script
	scriptPath := filepath.Join("scripts", "run_spider.py")
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		logger.WithError(err).Error("Failed to get absolute path to spider script")
	}

	// Initialize geocoder
	geocoder := geocoding.NewGeocoder(logger, "")

	// Initialize telegram service
	telegramService := telegram.NewService(logger)
	telegramService.SetDatabase(db)

	return &SpiderManager{
		logger:          logger,
		scriptPath:      absPath,
		db:              db,
		geocoder:        geocoder,
		telegramService: telegramService,
	}
}

// RunSpider executes a spider with the given parameters
// Place parameter must be normalized (lowercase, hyphenated, special cases handled)
func (m *SpiderManager) RunSpider(params SpiderParams) error {
	m.logger.WithFields(logrus.Fields{
		"spider_type": params.SpiderType,
		"place":       params.Place, // Already normalized by scheduler
		"max_pages":   params.MaxPages,
	}).Info("Starting spider")

	// Prepare the command
	cmd := exec.Command("python3", m.scriptPath)

	// Prepare input data
	input := map[string]interface{}{
		"spider_type": params.SpiderType,
		"place":       params.Place,
		"max_pages":   params.MaxPages,
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

		// Log raw output for debugging
		m.logger.WithField("raw_output", string(line)).Debug("Raw spider output")

		// First try parsing as a spider message
		var message SpiderMessage
		if err := json.Unmarshal(line, &message); err == nil && message.Type != "" {
			switch message.Type {
			case "items":
				// Process scraped items one by one
				var items []map[string]interface{}
				if err := json.Unmarshal(message.Data, &items); err != nil {
					m.logger.WithError(err).Error("Failed to parse items data")
					continue
				}
				m.logger.WithField("items_count", len(items)).Info("Received items from spider")

				// Process each item individually
				var newProperties []map[string]interface{}
				for _, item := range items {
					processedItems, err := m.db.InsertProperties([]map[string]interface{}{item})
					if err != nil {
						m.logger.WithError(err).Error("Failed to store property")
						continue
					}
					if len(processedItems) > 0 {
						newProperties = append(newProperties, processedItems[0])
					}
				}

				// After processing all items, handle geocoding and notifications
				if len(newProperties) > 0 {
					// Trigger geocoding in a background goroutine
					go func() {
						m.logger.Info("Starting geocoding for newly inserted properties...")
						if err := m.db.UpdateMissingCoordinates(m.geocoder); err != nil {
							m.logger.WithError(err).Error("Failed to update coordinates for new properties")
						}
					}()

					// Send notifications for new properties
					config, err := m.db.GetTelegramConfig()
					if err != nil {
						m.logger.WithError(err).Error("Failed to get Telegram config")
					} else if config != nil {
						m.telegramService.UpdateConfig(config)
						for _, prop := range newProperties {
							if err := m.telegramService.NotifyNewProperty(prop); err != nil {
								m.logger.WithError(err).Error("Failed to send Telegram notification")
							}
						}
					}
				}

			case "error":
				var errorData map[string]interface{}
				if err := json.Unmarshal(message.Data, &errorData); err != nil {
					m.logger.WithError(err).Error("Failed to parse error data")
					continue
				}
				m.logger.WithField("error", errorData).Error("Spider error")
			}
			continue
		}

		// If not a spider message, try parsing as a log message
		var logMessage struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
			Time  string `json:"time"`
		}
		if err := json.Unmarshal(line, &logMessage); err == nil && logMessage.Level != "" {
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

		// If we can't parse it as either message type, just log it as debug
		m.logger.Debug(string(line))
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
	params := SpiderParams{
		SpiderType: "active",
		Place:      place,
		MaxPages:   maxPages,
	}
	return m.RunSpider(params)
}

// RunSoldSpider runs the sold listings spider
func (m *SpiderManager) RunSoldSpider(place string, maxPages *int) error {
	params := SpiderParams{
		SpiderType: "sold",
		Place:      place,
		MaxPages:   maxPages,
	}
	return m.RunSpider(params)
}

// RunRefreshSpider runs the spider to refresh active listings and mark inactive ones
func (m *SpiderManager) RunRefreshSpider(place string) error {
	m.logger.WithField("place", place).Info("Starting refresh spider")

	// Run the active spider to collect current URLs
	params := SpiderParams{
		SpiderType: "refresh",
		Place:      place,
	}

	if err := m.RunSpider(params); err != nil {
		return fmt.Errorf("failed to run refresh spider: %v", err)
	}

	return nil
}

func (m *SpiderManager) runSpider(params SpiderParams) error {
	// Convert params to JSON
	jsonData := map[string]interface{}{
		"spider_type": params.SpiderType,
		"place":       params.Place,
	}
	if params.MaxPages != nil {
		jsonData["max_pages"] = *params.MaxPages
	}

	// ... rest of the function
	return nil
}

package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"fundamental/server/internal/database"
	"fundamental/server/internal/models"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Service struct {
	logger *logrus.Logger
	client *http.Client
	config *models.TelegramConfig
	db     *database.Database
}

func NewService(logger *logrus.Logger) *Service {
	return &Service{
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) UpdateConfig(config *models.TelegramConfig) {
	s.config = config
}

func (s *Service) SetDatabase(db *database.Database) {
	s.db = db
}

// getPriceAnalysis returns the price analysis for a property
func (s *Service) getPriceAnalysis(price, livingArea float64, postalCode string) (string, string, error) {
	if s.db == nil {
		return "", "", errors.New("database connection not initialized")
	}

	if livingArea <= 0 {
		return "", "", errors.New("invalid living area")
	}

	pricePerSqm := price / livingArea
	district := postalCode[:4]

	medianPrice, err := s.db.GetDistrictMedianPricePerSqm(district)
	if err != nil {
		return fmt.Sprintf("€%.0f/m²", pricePerSqm), "District comparison unavailable", err
	}

	if medianPrice <= 0 {
		return fmt.Sprintf("€%.0f/m²", pricePerSqm), "No recent sales in district", nil
	}

	priceDiff := ((pricePerSqm - medianPrice) / medianPrice) * 100
	var analysis string
	switch {
	case priceDiff <= -10:
		analysis = fmt.Sprintf("%.1f%% below district median (€%.0f/m²)", -priceDiff, medianPrice)
	case priceDiff >= 10:
		analysis = fmt.Sprintf("%.1f%% above district median (€%.0f/m²)", priceDiff, medianPrice)
	default:
		analysis = fmt.Sprintf("Close to district median (€%.0f/m²)", medianPrice)
	}

	return fmt.Sprintf("€%.0f/m²", pricePerSqm), analysis, nil
}

// SendMessage sends a message to the configured Telegram chat
func (s *Service) SendMessage(message string) error {
	if !s.config.IsEnabled {
		return nil
	}

	if s.config.BotToken == "" {
		return errors.New("Telegram bot token is not configured")
	}

	if s.config.ChatID == "" {
		return errors.New("Telegram chat ID is not configured")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.config.BotToken)
	payload := map[string]interface{}{
		"chat_id":    s.config.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message to Telegram API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return errors.New("invalid bot token - please check your token from @BotFather")
		case http.StatusBadRequest:
			return fmt.Errorf("invalid chat ID or message format: %s", string(body))
		case http.StatusForbidden:
			return errors.New("bot was blocked by the user or chat")
		case http.StatusNotFound:
			return errors.New("bot not found - please check your token from @BotFather")
		default:
			return fmt.Errorf("Telegram API error (status %d): %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// NotifyNewProperty sends a notification about a new property
func (s *Service) NotifyNewProperty(property map[string]interface{}) error {
	if !s.config.IsEnabled {
		return nil
	}

	if s.config.BotToken == "" {
		return errors.New("Telegram bot token is not configured")
	}

	if s.config.ChatID == "" {
		return errors.New("Telegram chat ID is not configured")
	}

	// Safely convert numeric values
	var price float64
	var livingArea float64

	// Handle price conversion
	switch p := property["price"].(type) {
	case int:
		price = float64(p)
	case float64:
		price = p
	default:
		s.logger.WithField("price", property["price"]).Error("Invalid price type")
		price = 0
	}

	// Handle living area conversion
	switch la := property["living_area"].(type) {
	case int:
		livingArea = float64(la)
	case float64:
		livingArea = la
	default:
		s.logger.WithField("living_area", property["living_area"]).Error("Invalid living area type")
		livingArea = 0
	}

	postalCode, ok := property["postal_code"].(string)
	if !ok {
		s.logger.Error("Invalid or missing postal code")
		postalCode = "Unknown"
	}

	var pricePerSqm string
	var priceAnalysis string

	// Only attempt price analysis if we have a valid database connection and valid data
	if s.db != nil && price > 0 && livingArea > 0 && postalCode != "Unknown" {
		var err error
		pricePerSqm, priceAnalysis, err = s.getPriceAnalysis(price, livingArea, postalCode)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get price analysis")
			pricePerSqm = "N/A"
			priceAnalysis = "N/A"
		}
	} else {
		pricePerSqm = fmt.Sprintf("€%.0f/m²", price/livingArea)
		priceAnalysis = "N/A (price analysis unavailable)"
	}

	// Format the message with property details
	title := "<b>New Property Listed!</b>"
	if property["status"] == "republished" {
		var republishCount int
		switch rc := property["republish_count"].(type) {
		case int:
			republishCount = rc
		case float64:
			republishCount = int(rc)
		default:
			republishCount = 1
		}

		if republishCount > 1 {
			title = fmt.Sprintf("<b>⚡ Property Republished! (%d times)</b>", republishCount)
		} else {
			title = "<b>⚡ Property Republished!</b>"
		}
	}

	// Safely handle year_built and num_rooms
	var yearBuilt interface{} = "N/A"
	if yb := property["year_built"]; yb != nil {
		switch v := yb.(type) {
		case int:
			yearBuilt = v
		case float64:
			yearBuilt = int(v)
		}
	}

	var numRooms interface{} = "N/A"
	if nr := property["num_rooms"]; nr != nil {
		switch v := nr.(type) {
		case int:
			numRooms = v
		case float64:
			numRooms = int(v)
		}
	}

	street, _ := property["street"].(string)
	city, _ := property["city"].(string)
	url, _ := property["url"].(string)

	message := fmt.Sprintf(
		"%s\n\n"+
			"🏠 %s\n"+
			"📍 %s, %s\n"+
			"💰 €%d\n"+
			"📐 %v m²\n"+
			"💵 %s\n"+
			"📊 %s\n"+
			"🏗️ Built: %v\n"+
			"🚪 Rooms: %v\n\n"+
			"🔗 <a href=\"%s\">View on Funda</a>",
		title,
		street,
		city,
		postalCode,
		int(price),
		livingArea,
		pricePerSqm,
		priceAnalysis,
		yearBuilt,
		numRooms,
		url,
	)

	return s.SendMessage(message)
}

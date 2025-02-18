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
func (s *Service) getPriceAnalysis(price float64, livingArea float64, postalCode string) (string, string, error) {
	if livingArea <= 0 || price <= 0 {
		return "", "", fmt.Errorf("invalid price or living area")
	}

	pricePerSqm := price / livingArea
	district := postalCode[:4]

	medianPricePerSqm, err := s.db.GetDistrictMedianPricePerSqm(district)
	if err != nil {
		return "", "", err
	}

	if medianPricePerSqm <= 0 {
		return fmt.Sprintf("€%.0f/m²", pricePerSqm), "NO DATA", nil
	}

	percentageDiff := ((pricePerSqm - medianPricePerSqm) / medianPricePerSqm) * 100

	var rating string
	switch {
	case percentageDiff < -20:
		rating = "GREAT"
	case percentageDiff < -5:
		rating = "GOOD"
	case percentageDiff < 5:
		rating = "NORMAL"
	case percentageDiff < 20:
		rating = "BAD"
	default:
		rating = "HORRIBLE"
	}

	return fmt.Sprintf("€%.0f/m²", pricePerSqm),
		fmt.Sprintf("%s (%+.1f%% vs. district median)", rating, percentageDiff),
		nil
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

	price := float64(property["price"].(int))
	livingArea := float64(property["living_area"].(int))
	postalCode := property["postal_code"].(string)

	pricePerSqm, priceAnalysis, err := s.getPriceAnalysis(price, livingArea, postalCode)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get price analysis")
		pricePerSqm = "N/A"
		priceAnalysis = "N/A"
	}

	// Format the message with property details
	message := fmt.Sprintf(
		"<b>New Property Listed!</b>\n\n"+
			"🏠 %s\n"+
			"�� %s, %s\n"+
			"💰 €%d\n"+
			"📐 %v m²\n"+
			"💵 %s\n"+
			"📊 %s\n"+
			"🏗️ Built: %v\n"+
			"🚪 Rooms: %v\n\n"+
			"🔗 <a href=\"%s\">View on Funda</a>",
		property["street"],
		property["city"],
		property["postal_code"],
		property["price"],
		property["living_area"],
		pricePerSqm,
		priceAnalysis,
		property["year_built"],
		property["num_rooms"],
		property["url"],
	)

	return s.SendMessage(message)
}

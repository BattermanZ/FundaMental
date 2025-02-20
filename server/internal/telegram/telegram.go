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
	"strings"
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

	activeMedian, activeCount, soldMedian, soldCount, err := s.db.GetDistrictPriceAnalysis(district)
	if err != nil {
		return fmt.Sprintf("€%s/m²", formatNumber(pricePerSqm)), "District comparison unavailable", err
	}

	// Format the analysis message
	var analysis strings.Builder
	analysis.WriteString("📊 <u>District Analysis</u>\n")

	// Compare with active listings
	if activeMedian > 0 {
		ratio := pricePerSqm / activeMedian
		var rating string
		switch {
		case ratio <= 0.80:
			rating = "<b>GREAT</b>"
		case ratio <= 0.95:
			rating = "<b>GOOD</b>"
		case ratio <= 1.05:
			rating = "<b>NORMAL</b>"
		case ratio <= 1.20:
			rating = "<b>BAD</b>"
		default:
			rating = "<b>HORRIBLE</b>"
		}
		diff := ((ratio - 1) * 100)
		analysis.WriteString(fmt.Sprintf("Current listings (%d properties):\n%s (%+.1f%% vs. median)\n\n", activeCount, rating, diff))
	} else {
		analysis.WriteString("Current listings (0 properties):\nNo active listings for comparison\n\n")
	}

	// Compare with sold properties
	if soldMedian > 0 {
		ratio := pricePerSqm / soldMedian
		var rating string
		switch {
		case ratio <= 0.80:
			rating = "<b>GREAT</b>"
		case ratio <= 0.95:
			rating = "<b>GOOD</b>"
		case ratio <= 1.05:
			rating = "<b>NORMAL</b>"
		case ratio <= 1.20:
			rating = "<b>BAD</b>"
		default:
			rating = "<b>HORRIBLE</b>"
		}
		diff := ((ratio - 1) * 100)
		analysis.WriteString(fmt.Sprintf("Past year sales (%d properties):\n%s (%+.1f%% vs. median)", soldCount, rating, diff))
	} else {
		analysis.WriteString("Past year sales (0 properties):\nNo recent sales for comparison")
	}

	return fmt.Sprintf("€%s/m²", formatNumber(pricePerSqm)), analysis.String(), nil
}

// formatNumber adds thousand separators to a number
func formatNumber(num float64) string {
	parts := strings.Split(fmt.Sprintf("%.0f", num), ".")
	intPart := parts[0]
	var result []byte
	for i, j := len(intPart)-1, 0; i >= 0; i, j = i-1, j+1 {
		if j > 0 && j%3 == 0 {
			result = append([]byte{','}, result...)
		}
		result = append([]byte{intPart[i]}, result...)
	}
	return string(result)
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

	var priceAnalysis string

	// Only attempt price analysis if we have a valid database connection and valid data
	if s.db != nil && price > 0 && livingArea > 0 && postalCode != "Unknown" {
		var err error
		_, priceAnalysis, err = s.getPriceAnalysis(price, livingArea, postalCode)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get price analysis")
			priceAnalysis = "N/A"
		}
	} else {
		priceAnalysis = "N/A (price analysis unavailable)"
	}

	// Format the message with property details
	title := "<b>New Property Listed!</b>"
	var priceText string

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

		// Get previous price if available
		if id, ok := property["id"].(int64); ok && s.db != nil {
			if previousPrice, err := s.db.GetPreviousPrice(id); err == nil && previousPrice > 0 {
				priceDiff := price - float64(previousPrice)
				priceDiffPercent := (priceDiff / float64(previousPrice)) * 100
				var arrow string
				if priceDiff > 0 {
					arrow = "📈"
				} else {
					arrow = "📉"
				}
				priceText = fmt.Sprintf("💰 €%s (%s %+.1f%% from €%s)",
					formatNumber(price),
					arrow,
					priceDiffPercent,
					formatNumber(float64(previousPrice)))
			} else {
				priceText = fmt.Sprintf("💰 €%s", formatNumber(price))
			}
		} else {
			priceText = fmt.Sprintf("💰 €%s", formatNumber(price))
		}
	} else {
		priceText = fmt.Sprintf("💰 €%s", formatNumber(price))
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
			"%s\n"+
			"📐 %v m²\n"+
			"💵 €%s/m²\n"+
			"🏗️ Built: %v\n"+
			"🚪 Rooms: %v\n"+
			"⚡ Energy label: %v\n\n"+
			"%s\n\n"+
			"🔗 <a href=\"%s\">View on Funda</a>",
		title,
		street,
		city,
		postalCode,
		priceText,
		livingArea,
		formatNumber(price/livingArea),
		yearBuilt,
		numRooms,
		property["energy_label"],
		priceAnalysis,
		url,
	)

	return s.SendMessage(message)
}

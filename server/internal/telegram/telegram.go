package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"fundamental/server/internal/models"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Service struct {
	logger *logrus.Logger
	client *http.Client
	config *models.TelegramConfig
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

func (s *Service) SendMessage(text string) error {
	if s.config == nil || !s.config.IsEnabled {
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.config.BotToken)

	message := struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}{
		ChatID:    s.config.ChatID,
		Text:      text,
		ParseMode: "HTML",
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *Service) NotifyNewProperty(property map[string]interface{}) error {
	if s.config == nil || !s.config.IsEnabled {
		return nil
	}

	// Format the message with property details
	message := fmt.Sprintf(
		"<b>New Property Listed!</b>\n\n"+
			"🏠 %s\n"+
			"📍 %s, %s\n"+
			"💰 €%d\n"+
			"🏗️ Built: %v\n"+
			"📐 Area: %v m²\n"+
			"🚪 Rooms: %v\n\n"+
			"🔗 <a href=\"%s\">View on Funda</a>",
		property["street"],
		property["city"],
		property["postal_code"],
		property["price"],
		property["year_built"],
		property["living_area"],
		property["num_rooms"],
		property["url"],
	)

	return s.SendMessage(message)
}

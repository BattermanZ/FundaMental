package models

import "time"

// TelegramConfig stores the bot credentials and basic settings
type TelegramConfig struct {
	ID        int64     `json:"id"`
	IsEnabled bool      `json:"is_enabled"`
	BotToken  string    `json:"bot_token"`
	ChatID    string    `json:"chat_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TelegramConfigRequest is used when updating the configuration
type TelegramConfigRequest struct {
	IsEnabled bool   `json:"is_enabled"`
	BotToken  string `json:"bot_token"`
	ChatID    string `json:"chat_id"`
}

// TelegramFilters stores the notification filter settings
type TelegramFilters struct {
	MinPrice      *int     `json:"min_price"`
	MaxPrice      *int     `json:"max_price"`
	MinLivingArea *int     `json:"min_living_area"`
	MaxLivingArea *int     `json:"max_living_area"`
	MinRooms      *int     `json:"min_rooms"`
	MaxRooms      *int     `json:"max_rooms"`
	Districts     []string `json:"districts"`
	EnergyLabels  []string `json:"energy_labels"`
}

// IsPropertyAllowed checks if a property matches the filter criteria
func (f *TelegramFilters) IsPropertyAllowed(property *Property) bool {
	if f == nil {
		return true // No filters means allow all
	}

	// Check price range
	if f.MinPrice != nil && (property.Price < *f.MinPrice) {
		return false
	}
	if f.MaxPrice != nil && (property.Price > *f.MaxPrice) {
		return false
	}

	// Check living area range
	if property.LivingArea != nil {
		if f.MinLivingArea != nil && (*property.LivingArea < *f.MinLivingArea) {
			return false
		}
		if f.MaxLivingArea != nil && (*property.LivingArea > *f.MaxLivingArea) {
			return false
		}
	} else if f.MinLivingArea != nil || f.MaxLivingArea != nil {
		return false // Filter requires living area but property has none
	}

	// Check number of rooms
	if property.NumRooms != nil {
		if f.MinRooms != nil && (*property.NumRooms < *f.MinRooms) {
			return false
		}
		if f.MaxRooms != nil && (*property.NumRooms > *f.MaxRooms) {
			return false
		}
	} else if f.MinRooms != nil || f.MaxRooms != nil {
		return false // Filter requires rooms but property has none
	}

	// Check district (postal code prefix)
	if len(f.Districts) > 0 {
		postalPrefix := property.PostalCode[:4] // First 4 digits of postal code
		allowed := false
		for _, district := range f.Districts {
			if district == postalPrefix {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check energy label
	if len(f.EnergyLabels) > 0 {
		if property.EnergyLabel == "" {
			return false
		}
		allowed := false
		for _, label := range f.EnergyLabels {
			if label == property.EnergyLabel {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

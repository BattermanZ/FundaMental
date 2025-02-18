package models

import "time"

type TelegramConfig struct {
	ID        int64     `json:"id"`
	BotToken  string    `json:"bot_token"`
	ChatID    string    `json:"chat_id"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TelegramConfigRequest struct {
	BotToken  string `json:"bot_token" binding:"required"`
	ChatID    string `json:"chat_id" binding:"required"`
	IsEnabled bool   `json:"is_enabled"`
}

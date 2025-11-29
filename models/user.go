package models

import "time"

type User struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Username   string    `gorm:"unique" json:"username"`
	Password   string    `json:"-"`
	TelegramID int64     `gorm:"unique" json:"telegram_id"` // Baru: ID Chat Telegram
	Role       string    `json:"role"`                      // Baru: 'admin' atau 'user'
	CreatedAt  time.Time `json:"created_at"`
}
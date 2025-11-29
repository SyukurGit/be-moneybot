package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"unique" json:"username"`
	Password     string    `json:"-"`
	TelegramID   int64     `gorm:"unique" json:"telegram_id"`
	Role         string    `json:"role"`
	
	// FITUR BARU: Budgeting
	DailyLimit   int       `json:"daily_limit"`   // Batas harian (cth: 100000)
	AlertMessage string    `json:"alert_message"` // Pesan (cth: "Woy boros!")
	
	CreatedAt    time.Time `json:"created_at"`
}
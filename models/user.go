package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"unique" json:"username"`
	Password     string    `json:"-"`
	
	// UBAH: Jadi Pointer (*int64) supaya bisa NULL (kosong) kalau user belum connect Telegram
	TelegramID   *int64    `gorm:"unique" json:"telegram_id"` 
	
	Role         string    `json:"role"` // 'admin' atau 'user'
	
	// FITUR BARU: Status User & Trial
	// Values: 'trial', 'suspended', 'active'
	Status       string    `json:"status" gorm:"default:'trial'"`
	TrialEndsAt  time.Time `json:"trial_ends_at"` // Kapan trial berakhir

	// Settingan Budget (Fitur Lama)
	DailyLimit   int       `json:"daily_limit"`
	AlertMessage string    `json:"alert_message"`
	
	CreatedAt    time.Time `json:"created_at"`
}
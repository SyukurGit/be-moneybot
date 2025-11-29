package models

import "time"

type Transaction struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`   // Baru: Penanda pemilik data
	Amount    int       `json:"amount"`
	Type      string    `json:"type"`
	Category  string    `json:"category"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	
	// Optional: Relasi ke User (biar GORM tahu)
	User User `gorm:"foreignKey:UserID" json:"-"`
}
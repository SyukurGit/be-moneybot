package models

import "time"

type PaymentLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `json:"user_id"`
	Username       string    `json:"username"` // Snapshot username saat upload
	ImagePath      string    `json:"image_path"`
	DetectedBank   string    `json:"detected_bank"`
	DetectedAmount int64     `json:"detected_amount"`
	RawOCRResponse string    `json:"raw_ocr_response"` // Simpan semua teks hasil bacaan OCR
	CreatedAt      time.Time `json:"created_at"`
	
	// Relasi
	User User `gorm:"foreignKey:UserID" json:"-"`
}
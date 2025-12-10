package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"net/http"
"golang.org/x/crypto/bcrypt"
	"github.com/gin-gonic/gin"
)

// Helper untuk ambil ID dari Token (sama seperti di transaction.go)
func getUserIDFromContext(c *gin.Context) uint {
	id, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return id.(uint)
}

// GET Settings (Untuk ditampilkan di form frontend nanti)
func GetUserSettings(c *gin.Context) {
	userID := getUserIDFromContext(c)
	var user models.User
	
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"daily_limit":   user.DailyLimit,
		"alert_message": user.AlertMessage,
	})
}

// UPDATE Settings (Simpan Limit & Pesan)
func UpdateUserSettings(c *gin.Context) {
	userID := getUserIDFromContext(c)
	var user models.User

	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var input struct {
		DailyLimit   int    `json:"daily_limit"`
		AlertMessage string `json:"alert_message"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data invalid"})
		return
	}

	// Update Data
	user.DailyLimit = input.DailyLimit
	user.AlertMessage = input.AlertMessage

	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Pengaturan budget berhasil disimpan!"})
}


// ... import yang sudah ada ...
// Tambahkan "golang.org/x/crypto/bcrypt" di import jika belum ada (biasanya sudah ada di auth.go, pindahkan ke import global atau biarkan jika auto-import)

// Struct Input Khusus Update Profil
type UpdateProfileInput struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TelegramID *int64 `json:"telegram_id"`
}

// ENDPOINT: PUT /api/user/profile
func UpdateUserProfile(c *gin.Context) {
	userID := getUserIDFromContext(c)
	var user models.User

	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// 1. Update Username (Cek duplikat otomatis handled by Gorm Unique Index)
	if input.Username != "" {
		user.Username = input.Username
	}

	// 2. Update Password (Hash dulu)
	if input.Password != "" {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		user.Password = string(hashed)
	}

	// 3. Update Telegram ID (Bisa diset angka atau null/0)
    // Kita cek field input, jika user mengirim field telegram_id di JSON
	if input.TelegramID != nil {
		user.TelegramID = input.TelegramID
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal update. Username/TelegramID mungkin sudah dipakai orang lain."})
		return
	}

	// Kembalikan data user terbaru agar frontend bisa update localStorage
	c.JSON(http.StatusOK, gin.H{
		"message": "Profil berhasil diperbarui!",
		"user": gin.H{
			"username":      user.Username,
			"status":        user.Status,
			"telegram_id":   user.TelegramID,
			"trial_ends_at": user.TrialEndsAt,
		},
	})
}
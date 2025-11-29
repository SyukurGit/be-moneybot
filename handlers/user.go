package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"net/http"

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
package handlers

import (
	"net/http"
	"backend-gin/database"
	"backend-gin/models"
	"backend-gin/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username atau password salah"})
		return
	}

	// UPDATE: Masukkan role user ke dalam token
	token, _ := utils.GenerateToken(user.ID, user.Role)
	
	// Kirim balik data user juga (biar frontend tahu dia admin/bukan)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"role": user.Role, 
		"username": user.Username,
	})
}

func RegisterAdmin(c *gin.Context) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	
	// Ganti dengan ID Telegram ASLI kamu
	user := models.User{
		Username:   "admin",
		Password:   string(hashedPassword),
		Role:       "admin", 
		TelegramID: 5321617875, 
	}
	
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Gagal membuat admin.",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Super Admin berhasil dibuat!"})
}
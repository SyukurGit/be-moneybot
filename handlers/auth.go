package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"backend-gin/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Struct untuk Validasi Input Login
type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Struct untuk Validasi Input Register
type RegisterInput struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data salah", "details": err.Error()})
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

	// Buat Token JWT
	token, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
		return
	}

	// Kirim Response Lengkap (termasuk status trial)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"role":          user.Role,
			"status":        user.Status,
			"trial_ends_at": user.TrialEndsAt,
		},
	})
}

func Register(c *gin.Context) {
	var input RegisterInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap", "details": err.Error()})
		return
	}

	// 1. Cek Konfirmasi Password
	if input.Password != input.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Konfirmasi password tidak cocok!"})
		return
	}

	// 2. Hash Password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	// 3. Siapkan Data User Baru (Mode Trial 1 Hari)
	newUser := models.User{
		Username:    input.Username,
		Password:    string(hashedPassword),
		Role:        "user", // Default user biasa
		Status:      "trial",
		TrialEndsAt: time.Now().Add(24 * time.Hour), // Trial 24 Jam dari sekarang
		TelegramID:  nil,                            // Belum bind telegram
	}

	// 4. Simpan ke Database
	if err := database.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah dipakai!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Registrasi berhasil! Mode Trial aktif selama 24 jam.",
		"user": gin.H{
			"username":      newUser.Username,
			"status":        newUser.Status,
			"trial_ends_at": newUser.TrialEndsAt,
		},
	})



	// ... kode Login & Register yang lama ...


}

// FUNGSI KHUSUS: Buat Super Admin (Hanya bisa sekali pakai atau pakai Secret Key)
func RegisterOwner(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Secret   string `json:"secret" binding:"required"` // Kunci Pengaman
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap", "details": err.Error()})
		return
	}

	// 1. Cek Kunci Rahasia (Hardcode di sini biar simpel, atau ambil dari .env)
	// Pastikan secret ini SAMA dengan yang kamu kirim di Postman nanti
	if input.Secret != "syukur_owner_2025" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Kunci rahasia salah! Anda bukan owner."})
		return
	}

	// 2. Hash Password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	// 3. Buat User dengan Level Tertinggi
	superAdmin := models.User{
		Username:    input.Username,
		Password:    string(hashedPassword),
		Role:        "admin",   // <--- SUPER ADMIN
		Status:      "active",  // <--- LANGSUNG AKTIF (Bebas Bayar)
		TrialEndsAt: time.Now().AddDate(100, 0, 0), // Aktif 100 Tahun
		TelegramID:  nil,       // Biarkan kosong dulu
	}

	// 4. Simpan ke Database
	if err := database.DB.Create(&superAdmin).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal buat admin. Username mungkin sudah ada."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "👑 Super Admin berhasil dibuat!",
		"data": gin.H{
			"username": superAdmin.Username,
			"role":     superAdmin.Role,
			"status":   superAdmin.Status,
		},
	})
}
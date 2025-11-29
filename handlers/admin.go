package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Helper: Cek apakah yang request adalah Admin?
func isAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	return exists && role == "admin"
}

// 1. LIST SEMUA USER (Hanya Admin)
func GetAllUsers(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak! Khusus Admin."})
		return
	}

	var users []models.User
	// Ambil semua user tapi sembunyikan passwordnya
	database.DB.Select("id, username, role, telegram_id, created_at").Find(&users)
	
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// 2. TAMBAH USER BARU (Hanya Admin)
func CreateUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak! Khusus Admin."})
		return
	}

	var input struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		TelegramID int64  `json:"telegram_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap"})
		return
	}

	// Hash Password user baru
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	newUser := models.User{
		Username:   input.Username,
		Password:   string(hashedPassword),
		TelegramID: input.TelegramID,
		Role:       "user", // Defaultnya user biasa
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal buat user (Username/Tele ID mungkin kembar)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User baru berhasil dibuat!", "data": newUser})
}

// 3. HAPUS USER (Hanya Admin)
func DeleteUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak! Khusus Admin."})
		return
	}

	id := c.Param("id") // Ambil ID dari URL /admin/users/:id

	// Hapus User
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus user"})
		return
	}

	// Opsional: Hapus juga semua transaksi milik user tersebut agar database bersih
	database.DB.Where("user_id = ?", id).Delete(&models.Transaction{})

	c.JSON(http.StatusOK, gin.H{"message": "User dan datanya berhasil dihapus"})
}



// ... kode lama ...

// 4. GET DETAIL & STATS USER (Untuk Halaman Detail)
func GetUserStats(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak!"})
		return
	}

	userID := c.Param("id")

	// 1. Ambil Data User
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// 2. Hitung Statistik Bulan Ini
	var income, expense int
	now := time.Now()
	// Tgl 1 bulan ini
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Hitung Income
	database.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND type = 'income' AND created_at >= ?", userID, startOfMonth).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&income)

	// Hitung Expense
	database.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND type = 'expense' AND created_at >= ?", userID, startOfMonth).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&expense)

	// 3. Cek "Last Active" (Berdasarkan transaksi terakhir)
	var lastTrx models.Transaction
	var lastActive time.Time
	if err := database.DB.Where("user_id = ?", userID).Order("created_at desc").First(&lastTrx).Error; err == nil {
		lastActive = lastTrx.CreatedAt
	} else {
		lastActive = user.CreatedAt // Kalau belum ada transaksi, pakai tgl daftar
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"telegram_id": user.TelegramID,
			"role":        user.Role,
		},
		"stats": gin.H{
			"income":      income,
			"expense":     expense,
			"balance":     income - expense,
			"last_active": lastActive,
		},
	})
}

// 5. UPDATE USER (Ganti Username / Password)
func UpdateUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak!"})
		return
	}

	id := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Update field jika diisi
	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Password != "" {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		user.Password = string(hashedPassword)
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update user (Username mungkin sudah dipakai)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data user berhasil diperbarui!"})
}

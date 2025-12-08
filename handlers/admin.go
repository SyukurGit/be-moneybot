package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Helper: Cek Admin
func isAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	return exists && role == "admin"
}

// 1. LIST USER (Menampilkan Status & Sisa Trial)
func GetAllUsers(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak!"})
		return
	}

	var users []models.User
	database.DB.Select("id, username, role, status, trial_ends_at, telegram_id, created_at").Find(&users)
	
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// 2. CREATE USER (Versi Admin: Otomatis ACTIVE / Bebas Bayar)
func CreateUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak!"})
		return
	}

	var input struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		TelegramID *int64 `json:"telegram_id"` // Optional
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap"})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	// Kalo Admin yang buat, anggap saja user VIP (langsung Active)
	newUser := models.User{
		Username:    input.Username,
		Password:    string(hashedPassword),
		Role:        "user",
		Status:      "active", // <--- LANGSUNG AKTIF
		TelegramID:  input.TelegramID,
		TrialEndsAt: time.Now().Add(365 * 24 * time.Hour), // Set setahun biar aman
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal buat user (Username/TeleID kembar)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User VIP berhasil dibuat!", "data": newUser})
}

// 3. DELETE USER
func DeleteUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak!"})
		return
	}

	id := c.Param("id")
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus"})
		return
	}
	// Hapus data transaksinya juga
	database.DB.Where("user_id = ?", id).Delete(&models.Transaction{})

	c.JSON(http.StatusOK, gin.H{"message": "User dihapus"})
}

// 4. GET USER STATS (Detail & Income/Expense)
func GetUserStats(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak!"})
		return
	}

	userID := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Hitung Statistik Bulan Ini
	var income, expense int
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	database.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND type = 'income' AND created_at >= ?", userID, startOfMonth).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&income)

	database.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND type = 'expense' AND created_at >= ?", userID, startOfMonth).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&expense)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"status":        user.Status,        // Info Status
			"trial_ends_at": user.TrialEndsAt,   // Info Expired
			"role":          user.Role,
		},
		"stats": gin.H{
			"income_this_month":  income,
			"expense_this_month": expense,
			"balance":            income - expense,
		},
	})
}

// 5. UPDATE DATA USER (Username/Pass)
// 5. UPDATE DATA USER (Username / Password / Telegram ID)
func UpdateUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak!"})
		return
	}

	id := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User hilang"})
		return
	}

	var input struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		TelegramID *int64 `json:"telegram_id"` // Tambahan baru
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Input salah"})
		return
	}

	// Update Field
	if input.Username != "" { user.Username = input.Username }
	if input.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		user.Password = string(hash)
	}
    // Update Telegram ID (Bisa diset ke angka baru atau null)
	if input.TelegramID != nil {
		user.TelegramID = input.TelegramID
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update (Username/TeleID mungkin kembar)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data user berhasil diperbarui!"})
}

// 6. [BARU] UPDATE STATUS & SUBSCRIPTION
// Endpoint: PATCH /api/admin/users/:id/status
func UpdateUserStatus(c *gin.Context) {
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
		Status       string `json:"status"`         // 'active', 'suspended', 'trial'
		AddTrialDays int    `json:"add_trial_days"` // +3 hari, +7 hari, dll
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data salah"})
		return
	}

	// 1. Logic Ganti Status
	if input.Status != "" {
		if input.Status == "active" || input.Status == "suspended" || input.Status == "trial" {
			user.Status = input.Status
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status harus: active / suspended / trial"})
			return
		}
	}

	// 2. Logic Tambah Hari Trial
	if input.AddTrialDays != 0 {
		// Jika trial sudah kadaluarsa, reset start dari sekarang
		if time.Now().After(user.TrialEndsAt) {
			user.TrialEndsAt = time.Now().Add(time.Duration(input.AddTrialDays) * 24 * time.Hour)
		} else {
			// Jika masih aktif, tambahkan harinya
			user.TrialEndsAt = user.TrialEndsAt.Add(time.Duration(input.AddTrialDays) * 24 * time.Hour)
		}
		
		// Kalau nambah hari, otomatis set status ke trial (kecuali admin minta lain)
		if input.Status == "" {
			user.Status = "trial"
		}
	}

	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "Status user berhasil diperbarui!",
		"result": gin.H{
			"username":      user.Username,
			"new_status":    user.Status,
			"trial_ends_at": user.TrialEndsAt,
		},
	})
}
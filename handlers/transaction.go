package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Helper: Ambil UserID dari Token
func getUserID(c *gin.Context) uint {
	id, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return id.(uint)
}

// Helper: Cek Limit Harian (Dipakai oleh Web & Bot)
func CheckDailyLimit(userID uint, currentAmount int) string {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return ""
	}

	if user.DailyLimit <= 0 {
		return ""
	}

	var totalToday int
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	database.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND type = 'expense' AND created_at >= ?", userID, startOfDay).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&totalToday)

	// Tambahkan transaksi yang baru saja akan diinput untuk pengecekan prediksi
	// (Opsional, atau cek total existing + new)
	if (totalToday + currentAmount) >= user.DailyLimit {
		pesan := user.AlertMessage
		if pesan == "" {
			pesan = "⚠️ <b>WARNING:</b> Kamu sudah melebihi budget harian!"
		}
		return pesan
	}

	return ""
}

// 1. GET ALL TRANSACTIONS (PAGINATION) - Tetap seperti sebelumnya
func GetTransactions(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	if page < 1 { page = 1 }
	if limit < 1 { limit = 10 }
	if limit > 100 { limit = 100 }

	offset := (page - 1) * limit

	query := database.DB.Model(&models.Transaction{}).Where("user_id = ?", userID)

	if tipe := c.Query("type"); tipe != "" {
		query = query.Where("type = ?", tipe)
	}

	if search := c.Query("search"); search != "" {
		search = "%" + search + "%"
		query = query.Where("category LIKE ? OR note LIKE ?", search, search)
	}

	var total int64
	query.Count(&total)

	query.Order("created_at desc").Limit(limit).Offset(offset).Find(&trx)

	c.JSON(http.StatusOK, gin.H{
		"data": trx,
		"meta": gin.H{
			"current_page": page,
			"limit":        limit,
			"total_data":   total,
			"total_pages":  math.Ceil(float64(total) / float64(limit)),
		},
	})
}

// 2. CREATE TRANSACTION (WEB INPUT) - FITUR BARU
func CreateTransaction(c *gin.Context) {
	userID := getUserID(c)

	// Gunakan struct khusus untuk menerima input string (biar bisa handle "100.000")
	var input struct {
		Type     string `json:"type" binding:"required"`     // income / expense
		Amount   string `json:"amount" binding:"required"`   // String: "100.000" atau "50000"
		Category string `json:"category" binding:"required"` // Gaji, Makan, dll
		Note     string `json:"note"`                        // Opsional
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Bersihkan format Rupiah (Hapus titik dan koma)
	cleanAmount := strings.ReplaceAll(input.Amount, ".", "")
	cleanAmount = strings.ReplaceAll(cleanAmount, ",", "")
	
	amountInt, err := strconv.Atoi(cleanAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format jumlah uang salah. Harusnya angka."})
		return
	}

	// Simpan
	trx := models.Transaction{
		UserID:   userID,
		Amount:   amountInt,
		Type:     input.Type,
		Category: input.Category,
		Note:     input.Note,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&trx).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data"})
		return
	}

	// Cek Alert Limit (Hanya return pesan warning, tidak error)
	alertMsg := ""
	if input.Type == "expense" {
		// Kita cek limit berdasarkan total HARI INI (termasuk yg baru diinput)
		// Note: helper CheckDailyLimit menghitung total di DB. 
		// Karena data baru SUDAH tersimpan barusan, maka CheckDailyLimit akan menghitungnya.
		// Namun helper saya tadi menghitung total yg ada di DB. 
		// Karena kita panggil create dulu baru cek, maka totalnya valid.
		// Tapi helper `CheckDailyLimit` saya modif sedikit logicnya di atas.
		// Mari kita panggil helpernya:
		alertMsg = CheckDailyLimit(userID, 0) // Param kedua 0 karena data sudah masuk DB
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil disimpan!",
		"data":    trx,
		"alert":   alertMsg,
	})
}

// 3. GET TODAY TRANSACTIONS (UNTUK TABEL BAWAH) - FITUR BARU
func GetTodayTransactions(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Ambil semua transaksi hari ini, urutkan dari yg terbaru
	database.DB.Where("user_id = ? AND created_at >= ?", userID, startOfDay).
		Order("created_at desc").
		Find(&trx)

	c.JSON(http.StatusOK, gin.H{"data": trx})
}

// 4. DELETE TRANSACTION (WEB DELETE) - FITUR BARU
func DeleteTransaction(c *gin.Context) {
	userID := getUserID(c)
	id := c.Param("id")

	// Pastikan user menghapus data miliknya sendiri
	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Transaction{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data tidak ditemukan atau bukan milikmu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaksi berhasil dihapus"})
}

// 5. GET SUMMARY
func GetSummary(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction
	database.DB.Where("user_id = ?", userID).Find(&trx)

	var income, expense int
	for _, t := range trx {
		if t.Type == "income" {
			income += t.Amount
		} else if t.Type == "expense" {
			expense += t.Amount
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_income":  income,
		"total_expense": expense,
		"balance":       income - expense,
	})
}

// 6. GET DAILY CHART
// 3. GET CHART DATA (Support Filter Bulan)
func GetDailyChart(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction
	
	// Default: Ambil 30 hari terakhir
	query := database.DB.Where("user_id = ?", userID)

	monthStr := c.Query("month")
	yearStr := c.Query("year")

	// Jika ada filter bulan & tahun
	if monthStr != "" && yearStr != "" {
		month, _ := strconv.Atoi(monthStr)
		year, _ := strconv.Atoi(yearStr)
		
		// Tanggal 1 bulan tersebut
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		// Tanggal 1 bulan berikutnya (batas atas)
		endDate := startDate.AddDate(0, 1, 0)

		query = query.Where("created_at >= ? AND created_at < ?", startDate, endDate)
	} else {
		// Fallback: 30 Hari Terakhir
		last30Days := time.Now().AddDate(0, 0, -30)
		query = query.Where("created_at >= ?", last30Days)
	}

	query.Order("created_at asc").Find(&trx)

	type DailyStats struct {
		Date    string `json:"date"`
		Income  int    `json:"income"`
		Expense int    `json:"expense"`
	}
	
	statsMap := make(map[string]*DailyStats)
	for _, t := range trx {
		// Format tanggal YYYY-MM-DD
		dateStr := t.CreatedAt.Format("2006-01-02")
		if _, exists := statsMap[dateStr]; !exists {
			statsMap[dateStr] = &DailyStats{Date: dateStr}
		}
		if t.Type == "income" {
			statsMap[dateStr].Income += t.Amount
		} else {
			statsMap[dateStr].Expense += t.Amount
		}
	}

	var result []DailyStats
	for _, v := range statsMap {
		result = append(result, *v)
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// 7. GET CATEGORIES
func GetCategorySummary(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction
	database.DB.Where("user_id = ?", userID).Find(&trx)

	type CatStats struct {
		Category string `json:"category"`
		Total    int    `json:"total"`
		Type     string `json:"type"`
	}

	tempMap := make(map[string]int)
	for _, t := range trx {
		key := t.Type + "-" + t.Category
		tempMap[key] += t.Amount
	}

	var results []CatStats
	for key, total := range tempMap {
		parts := strings.Split(key, "-")
		if len(parts) >= 2 {
			results = append(results, CatStats{
				Type:     parts[0],
				Category: parts[1],
				Total:    total,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
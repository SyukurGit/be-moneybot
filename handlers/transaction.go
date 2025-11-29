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

// Helper untuk ambil UserID dari Context
func getUserID(c *gin.Context) uint {
	id, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return id.(uint)
}

// 1. GET /api/transactions (DENGAN PAGINATION)
func GetTransactions(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction

	// 1. Ambil Parameter Page & Limit (Default: Page 1, Limit 10)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	// Cegah input negatif
	if page < 1 { page = 1 }
	if limit < 1 { limit = 10 }
	if limit > 100 { limit = 100 } // Batas maksimal biar server gak berat

	offset := (page - 1) * limit

	// 2. Siapkan Query Dasar
	query := database.DB.Model(&models.Transaction{}).Where("user_id = ?", userID)

	// Filter Tipe (Income/Expense) jika ada
	if tipe := c.Query("type"); tipe != "" {
		query = query.Where("type = ?", tipe)
	}

	// Filter Pencarian (Opsional: Search Category/Note)
	if search := c.Query("search"); search != "" {
		search = "%" + search + "%"
		query = query.Where("category LIKE ? OR note LIKE ?", search, search)
	}

	// 3. Hitung Total Data (Untuk Pagination)
	var total int64
	query.Count(&total)

	// 4. Ambil Data Sesuai Halaman (Limit & Offset)
	query.Order("created_at desc").Limit(limit).Offset(offset).Find(&trx)

	// 5. Kirim Respon dengan Metadata
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

// 2. GET /api/summary (Tetap Sama)
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

// 3. GET /api/chart/daily (Tetap Sama)
func GetDailyChart(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction
	last30Days := time.Now().AddDate(0, 0, -30)

	database.DB.Where("user_id = ? AND created_at >= ?", userID, last30Days).
		Order("created_at asc").
		Find(&trx)

	type DailyStats struct {
		Date    string `json:"date"`
		Income  int    `json:"income"`
		Expense int    `json:"expense"`
	}
	
	statsMap := make(map[string]*DailyStats)
	for _, t := range trx {
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

// 4. GET /api/categories (Tetap Sama)
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
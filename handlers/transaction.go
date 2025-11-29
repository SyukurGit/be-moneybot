package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Helper untuk ambil UserID dari Context (hasil kerja Middleware)
func getUserID(c *gin.Context) uint {
	id, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return id.(uint)
}

// 1. GET /api/transactions (Hanya punya user login)
func GetTransactions(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction

	// Filter UserID
	query := database.DB.Where("user_id = ?", userID).Order("created_at desc")

	if tipe := c.Query("type"); tipe != "" {
		query = query.Where("type = ?", tipe)
	}

	query.Find(&trx)
	c.JSON(http.StatusOK, gin.H{"data": trx})
}

// 2. GET /api/summary (Hanya punya user login)
func GetSummary(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction
	
	// Filter UserID
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

// 3. GET /api/chart/daily (Hanya punya user login)
func GetDailyChart(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction
	last30Days := time.Now().AddDate(0, 0, -30)

	// Filter UserID
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

// 4. GET /api/categories (Hanya punya user login)
func GetCategorySummary(c *gin.Context) {
	userID := getUserID(c)
	var trx []models.Transaction
	
	// Filter UserID
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
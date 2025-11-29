package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"strings" 

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// GET /api/export?month=11&year=2025
func ExportExcel(c *gin.Context) {
	userID := getUserID(c) // Helper dari transaction.go (pastikan package sama)

	// 1. Ambil Filter Bulan & Tahun (Opsional, default = semua)
	monthStr := c.Query("month")
	yearStr := c.Query("year")

	var trx []models.Transaction
	query := database.DB.Where("user_id = ?", userID).Order("created_at desc")

	// Jika ada filter bulan/tahun
	if monthStr != "" && yearStr != "" {
		month, _ := strconv.Atoi(monthStr)
		year, _ := strconv.Atoi(yearStr)
		
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		endDate := startDate.AddDate(0, 1, 0) // Awal bulan depan

		query = query.Where("created_at >= ? AND created_at < ?", startDate, endDate)
	}

	query.Find(&trx)

	// 2. Buat File Excel
	f := excelize.NewFile()
	sheetName := "Laporan Keuangan"
	f.SetSheetName("Sheet1", sheetName)

	// 3. Bikin Header (Baris 1)
	headers := []string{"No", "Tanggal", "Jam", "Tipe", "Kategori", "Catatan", "Jumlah (Rp)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// Style Header (Bold + Warna)
	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#4F46E5"}, Pattern: 1}, // Biru Keren
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheetName, "A1", "G1", styleHeader)

	// 4. Isi Data (Mulai Baris 2)
	row := 2
	for i, t := range trx {
		// Format Data
		dateStr := t.CreatedAt.Format("02-01-2006")
		timeStr := t.CreatedAt.Format("15:04")
		
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), dateStr)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), timeStr)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), strings.ToUpper(t.Type))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), t.Category)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), t.Note)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), t.Amount)

		// Warna Warni Tipe (Hijau Income, Merah Expense)
		if t.Type == "income" {
			styleIncome, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "#10B981"}}) // Hijau
			f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleIncome)
		} else {
			styleExpense, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "#EF4444"}}) // Merah
			f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styleExpense)
		}

		row++
	}

	// Auto Width (Biar kolom lebar sesuai isi)
	f.SetColWidth(sheetName, "A", "A", 5)  // No
	f.SetColWidth(sheetName, "B", "C", 15) // Tgl, Jam
	f.SetColWidth(sheetName, "D", "E", 15) // Tipe, Kategori
	f.SetColWidth(sheetName, "F", "F", 30) // Catatan
	f.SetColWidth(sheetName, "G", "G", 20) // Jumlah

	// 5. Kirim File ke Browser
	fileName := fmt.Sprintf("Laporan_Syukur_%s.xlsx", time.Now().Format("20060102"))
	
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Transfer-Encoding", "binary")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate excel"})
	}
}
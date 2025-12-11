package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend-gin/database"
	"backend-gin/models"

	"github.com/gin-gonic/gin"
)

// Struktur Respon dari API OCR.Space
type OCRResponse struct {
	ParsedResults []struct {
		ParsedText string `json:"ParsedText"`
	} `json:"ParsedResults"`
	OCRExitCode int `json:"OCRExitCode"`
}

// Struktur Hasil Analisa Kita
type AIResponse struct {
	IsValid bool   `json:"is_valid"`
	Amount  int64  `json:"amount"`
	Bank    string `json:"bank"`
	Reason  string `json:"reason"`
}

// Helper untuk memastikan folder uploads ada
func ensureUploadDir() string {
	path := "./uploads"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0755)
	}
	return path
}

// ---------------------------------------------------------
// 1. VERIFIKASI OTOMATIS (AUTO - OCR)
// ---------------------------------------------------------
func VerifyPayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File diperlukan"})
		return
	}
	defer file.Close()

	uploadDir := ensureUploadDir()

	// FIX NAMA FILE: Hapus spasi agar URL gambar tidak rusak
	cleanFilename := strings.ReplaceAll(header.Filename, " ", "_")
	filename := fmt.Sprintf("AUTO_%d_%d_%s", userID.(uint), time.Now().Unix(), cleanFilename)
	savePath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(header, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file"})
		return
	}

	// PROSES OCR KE API EKSTERNAL
	apiKey := os.Getenv("OCR_API_KEY")
	// Jika API Key kosong, anggap manual check (fallback aman)
	if apiKey == "" {
		savePaymentLog(userID.(uint), filename, "MANUAL_CHECK", 0, "API Key Missing", c)
		c.JSON(http.StatusAccepted, gin.H{"message": "OCR Off, masuk verifikasi manual", "manual_check": true})
		return
	}

	// ... Logika OCR ...
	var ocr OCRResponse
	// var errOCR error
	
	// Buka file lokal untuk dikirim
	savedFile, _ := os.Open(savePath)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", filename)
	io.Copy(fw, savedFile)
	savedFile.Close()
	w.WriteField("language", "eng")
	w.WriteField("OCREngine", "2")
	w.Close()

	req, _ := http.NewRequest("POST", "https://api.ocr.space/parse/image", &buf)
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	
	if err != nil {
		// Jika OCR Error/Timeout -> Lempar ke Manual
		savePaymentLog(userID.(uint), filename, "MANUAL_CHECK", 0, "OCR Timeout", c)
		c.JSON(http.StatusAccepted, gin.H{"message": "Gagal baca otomatis, masuk antrian admin.", "manual_check": true})
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &ocr)

	if ocr.OCRExitCode != 1 || len(ocr.ParsedResults) == 0 {
		// Jika OCR Gagal Baca -> Lempar ke Manual
		savePaymentLog(userID.(uint), filename, "MANUAL_CHECK", 0, "OCR Failed Read", c)
		c.JSON(http.StatusAccepted, gin.H{"message": "Struk tidak terbaca, masuk antrian admin.", "manual_check": true})
		return
	}

	text := strings.ToUpper(ocr.ParsedResults[0].ParsedText)
	result := extractPaymentInfo(text)

	// SIMPAN LOG
	// KUNCI: Jika Valid, simpan bank asli. Jika tidak, simpan deteksinya.
	savePaymentLog(userID.(uint), filename, result.Bank, result.Amount, text, c)

	if !result.IsValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bukti ditolak: " + result.Reason})
		return
	}

	// Aktifkan User
	var user models.User
	if err := database.DB.First(&user, userID).Error; err == nil {
		user.Status = "active"
		database.DB.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pembayaran Valid! Akun Aktif.", "data": result})
}

// ---------------------------------------------------------
// 2. VERIFIKASI MANUAL (MANUAL UPLOAD)
// ---------------------------------------------------------
func ManualPaymentUpload(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File wajib ada"})
		return
	}
	defer file.Close()

	uploadDir := ensureUploadDir()

	// FIX NAMA FILE: Tambah prefix MANUAL_ biar jelas
	cleanFilename := strings.ReplaceAll(header.Filename, " ", "_")
	filename := fmt.Sprintf("MANUAL_%d_%d_%s", userID.(uint), time.Now().Unix(), cleanFilename)
	savePath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(header, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal simpan file"})
		return
	}

	// Update Status User -> Pending
	var user models.User
	if err := database.DB.First(&user, userID).Error; err == nil {
		user.Status = "pending"
		database.DB.Save(&user)
	}

	// SIMPAN LOG DENGAN CAP KHUSUS: "MANUAL_CHECK"
	// Ini yang bikin gambar ini TAMPIL di Tab Manual Admin
	log := models.PaymentLog{
		UserID:         userID.(uint),
		Username:       user.Username,
		ImagePath:      "uploads/" + filename, // Path relatif untuk frontend
		DetectedBank:   "MANUAL_CHECK",        // <--- FLAG PENTING
		DetectedAmount: 0,
		RawOCRResponse: "User upload manual (Bypass AI)",
		CreatedAt:      time.Now(),
	}
	database.DB.Create(&log)

	c.JSON(http.StatusOK, gin.H{"message": "Terkirim ke Admin", "status": "pending"})
}

// HELPER SIMPAN LOG
func savePaymentLog(userID uint, filename, bank string, amount int64, rawText string, c *gin.Context) {
	var user models.User
	database.DB.First(&user, userID)

	paymentLog := models.PaymentLog{
		UserID:         user.ID,
		Username:       user.Username,
		ImagePath:      "uploads/" + filename, // Pastikan format ini konsisten
		DetectedBank:   bank,
		DetectedAmount: amount,
		RawOCRResponse: rawText,
		CreatedAt:      time.Now(),
	}
	database.DB.Create(&paymentLog)
}

// LOGIC EKSTRAKSI TEKS (Sama seperti sebelumnya)
func extractPaymentInfo(text string) AIResponse {
	r := AIResponse{}
	// Logic cek BERHASIL/SUCCESS
	if !strings.Contains(text, "BERHASIL") && !strings.Contains(text, "SUCCESS") && !strings.Contains(text, "SUKSES") {
		r.IsValid = false
		r.Reason = "Tidak ada kata BERHASIL/SUKSES"
		return r
	}
	// Logic cek Bank
	if strings.Contains(text, "BCA") { r.Bank = "BCA" } else 
	if strings.Contains(text, "DANA") { r.Bank = "DANA" } else 
	if strings.Contains(text, "GOPAY") { r.Bank = "GOPAY" } else 
	if strings.Contains(text, "MANDIRI") { r.Bank = "MANDIRI" } else 
	if strings.Contains(text, "BRI") { r.Bank = "BRI" } else { r.Bank = "Unknown" }

	// Logic cek Nominal (Simple)
	lines := strings.Fields(text)
	for _, part := range lines {
		cleanPart := strings.ReplaceAll(strings.ReplaceAll(part, ".", ""), ",", "")
		if strings.HasPrefix(cleanPart, "RP") {
			r.Amount = parseInt(strings.TrimPrefix(cleanPart, "RP"))
			if r.Amount > 0 { break }
		}
	}
	r.IsValid = true
	return r
}

func parseInt(s string) int64 {
	var x int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			x = x*10 + int64(c-'0')
		}
	}
	return x
}
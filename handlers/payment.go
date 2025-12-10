package handlers

import (
	"bytes"
	"encoding/json"
	
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"backend-gin/database"
	"backend-gin/models"
	"github.com/gin-gonic/gin"
)

type OCRResponse struct {
	ParsedResults []struct {
		ParsedText string `json:"ParsedText"`
	} `json:"ParsedResults"`
	OCRExitCode int `json:"OCRExitCode"`
}

type AIResponse struct {
	IsValid bool   `json:"is_valid"`
	Amount  int64  `json:"amount"`
	Bank    string `json:"bank"`
	Reason  string `json:"reason"`
}

func VerifyPayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// ambil file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File diperlukan"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maks 5MB"})
		return
	}

	// --- kirim ke OCR.SPACE ---
	apiKey := os.Getenv("OCR_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API Key kosong"})
		return
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, _ := w.CreateFormFile("file", header.Filename)
	io.Copy(fw, file)

	w.WriteField("language", "eng")
	w.WriteField("OCREngine", "2") // paling bagus free plan
	w.Close()

	req, _ := http.NewRequest("POST", "https://api.ocr.space/parse/image", &buf)
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Request OCR gagal: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var ocr OCRResponse
	if err := json.Unmarshal(body, &ocr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Parse OCR gagal"})
		return
	}

	// kalau OCR gagal
	if ocr.OCRExitCode != 1 || len(ocr.ParsedResults) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OCR tidak membaca teks"})
		return
	}

	text := strings.ToUpper(ocr.ParsedResults[0].ParsedText)

	// ------------------------------------
	//  BUAT LOGIKA DETEKSI VALIDASI STRUK
	// ------------------------------------
	result := extractPaymentInfo(text) // fungsi kita buat bawah

	if !result.IsValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ditolak: " + result.Reason})
		return
	}

	// update user aktif jika valid
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	user.Status = "active"
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "Akun aktif",
		"data":    result,
	})
}

// ---------------------------
// fungsi kecil untuk parsing
// ---------------------------
func extractPaymentInfo(text string) AIResponse {
	r := AIResponse{}

	// cek tanda struk berhasil
	if !strings.Contains(text, "BERHASIL") && !strings.Contains(text, "SUCCESS") {
		r.IsValid = false
		r.Reason = "Tidak ditemukan status sukses transfer"
		return r
	}

	// deteksi bank
	if strings.Contains(text, "BCA") {
		r.Bank = "BCA"
	}
	if strings.Contains(text, "DANA") {
		r.Bank = "DANA"
	}
	if strings.Contains(text, "GOPAY") {
		r.Bank = "GOPAY"
	}
	if r.Bank == "" {
		r.Bank = "Unknown"
	}

	// deteksi nominal (regex bisa ditambah lebih pintar)
	for _, part := range strings.Fields(text) {
		part = strings.ReplaceAll(part, ".", "")
		part = strings.ReplaceAll(part, ",", "")
		if strings.HasPrefix(part, "Rp") || strings.HasPrefix(part, "RP") {
			num := strings.TrimLeft(part, "RpRP")
			r.Amount = parseInt(num)
			break
		}
	}

	r.IsValid = true
	r.Reason = "OCR OK"
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

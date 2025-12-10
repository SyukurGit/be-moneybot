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

func VerifyPayment(c *gin.Context) {
	// 1. Ambil User ID dari Token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Ambil File dari Form Upload
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File diperlukan"})
		return
	}
	defer file.Close() // Tutup file multipart stream

	// Validasi Ukuran File (Maks 5MB)
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maksimal ukuran file 5MB"})
		return
	}

	// 3. SIMPAN FILE KE SERVER (Local Storage)
	// Buat folder uploads jika belum ada
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// Generate nama file unik: userID_timestamp_namaasli
	filename := fmt.Sprintf("%d_%d_%s", userID.(uint), time.Now().Unix(), filepath.Base(header.Filename))
	savePath := filepath.Join(uploadDir, filename)

	// Simpan file menggunakan helper Gin
	if err := c.SaveUploadedFile(header, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file gambar"})
		return
	}

	// 4. KIRIM FILE LOKAL KE OCR.SPACE DENGAN RETRY
	apiKey := os.Getenv("OCR_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API Key OCR belum disetting di .env"})
		return
	}

	var ocr OCRResponse
	var errOCR error
	maxRetries := 2 // Coba maksimal 2 kali

	for i := 0; i <= maxRetries; i++ {
		// Buka file yang baru saja disimpan untuk dikirim ke OCR
		// Kita buka ulang setiap retry untuk memastikan pointer file di awal
		savedFile, err := os.Open(savePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file tersimpan"})
			return
		}
		
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)

		// Masukkan file ke form-data request
		fw, _ := w.CreateFormFile("file", filename)
		io.Copy(fw, savedFile)
		savedFile.Close() // Tutup file setelah dicopy ke buffer

		// Setting parameter OCR
		w.WriteField("language", "eng")
		w.WriteField("OCREngine", "2") // Engine 2 biasanya lebih bagus untuk angka
		w.Close()

		// Kirim Request ke OCR.Space
		req, _ := http.NewRequest("POST", "https://api.ocr.space/parse/image", &buf)
		req.Header.Set("apikey", apiKey)
		req.Header.Set("Content-Type", w.FormDataContentType())

		// TIMEOUT DITINGKATKAN KE 60 DETIK
		client := &http.Client{Timeout: 60 * time.Second} 
		resp, err := client.Do(req)
		
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			
			// Coba parse JSON
			if jsonErr := json.Unmarshal(body, &ocr); jsonErr == nil {
				// Jika sukses parsing dan OCR exit code valid, break loop
				if ocr.OCRExitCode == 1 && len(ocr.ParsedResults) > 0 {
					errOCR = nil
					break
				}
			}
		}
		
		errOCR = fmt.Errorf("Percobaan ke-%d gagal: %v", i+1, err)
		time.Sleep(1 * time.Second) // Tunggu 1 detik sebelum retry
	}

	// Jika setelah retry masih gagal
	if errOCR != nil || ocr.OCRExitCode != 1 || len(ocr.ParsedResults) == 0 {
		// Fallback: Jangan gagalkan total, tapi beri info manual check
		// Hapus kode error keras, kita log saja sebagai manual check
		c.JSON(http.StatusAccepted, gin.H{
			"message": "Upload berhasil, namun OCR lambat merespon. Admin akan verifikasi manual.",
			"manual_check": true,
		})
		// Tetap simpan log tapi dengan status 'Pending Manual'
		savePaymentLog(userID.(uint), filename, "FAILED_TIMEOUT", 0, "OCR Timeout/Error", c)
		return
	}

	text := strings.ToUpper(ocr.ParsedResults[0].ParsedText)

	// 6. LOGIKA VALIDASI STRUK
	result := extractPaymentInfo(text)

	// 7. SIMPAN LOG & UPDATE USER
	savePaymentLog(userID.(uint), filename, result.Bank, result.Amount, text, c)

	// 8. KEPUTUSAN FINAL
	if !result.IsValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bukti ditolak: " + result.Reason})
		return
	}

	// Jika Valid, Aktifkan User
	var user models.User
	if err := database.DB.First(&user, userID).Error; err == nil {
		user.Status = "active"
		database.DB.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pembayaran Valid! Akun Anda telah aktif.",
		"data":    result,
	})
}

// Helper untuk simpan log agar kode VerifyPayment lebih bersih
func savePaymentLog(userID uint, filename, bank string, amount int64, rawText string, c *gin.Context) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return
	}

	paymentLog := models.PaymentLog{
		UserID:         user.ID,
		Username:       user.Username,
		ImagePath:      "uploads/" + filename,
		DetectedBank:   bank,
		DetectedAmount: amount,
		RawOCRResponse: rawText,
		CreatedAt:      time.Now(),
	}
	database.DB.Create(&paymentLog)
}

// ---------------------------
// HELPER FUNCTIONS (Tetap sama)
// ---------------------------

func extractPaymentInfo(text string) AIResponse {
	r := AIResponse{}

	// Cek kata kunci kesuksesan transfer
	if !strings.Contains(text, "BERHASIL") && !strings.Contains(text, "SUCCESS") && !strings.Contains(text, "SUKSES") {
		r.IsValid = false
		r.Reason = "Tidak ditemukan status sukses transfer pada struk"
		return r
	}

	// Deteksi Bank
	if strings.Contains(text, "BCA") {
		r.Bank = "BCA"
	} else if strings.Contains(text, "DANA") {
		r.Bank = "DANA"
	} else if strings.Contains(text, "GOPAY") {
		r.Bank = "GOPAY"
	} else if strings.Contains(text, "MANDIRI") {
		r.Bank = "MANDIRI"
	} else if strings.Contains(text, "BRI") {
		r.Bank = "BRI"
	} else {
		r.Bank = "Unknown" // Tetap valid, tapi bank tidak dikenal
	}

	// Deteksi Nominal (Logic Parsing Sederhana)
	// Mencari string yang diawali Rp/RP
	lines := strings.Fields(text)
	for _, part := range lines {
		// Bersihkan titik dan koma
		cleanPart := strings.ReplaceAll(part, ".", "")
		cleanPart = strings.ReplaceAll(cleanPart, ",", "")

		if strings.HasPrefix(cleanPart, "RP") {
			// Hapus prefix RP
			numStr := strings.TrimPrefix(cleanPart, "RP")
			// Parsing ke int64
			amount := parseInt(numStr)
			if amount > 0 {
				r.Amount = amount
				break // Ambil angka pertama yang valid
			}
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
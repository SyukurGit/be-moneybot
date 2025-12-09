package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

"backend-gin/database"
    "backend-gin/models"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GANTI INI DENGAN API KEY GOOGLE AI KAMU
const GEMINI_API_KEY = "AIzaSyDzZPiPk-zTNz52eXbuAgcuPdgDhOptYU8"
// Struktur respon dari AI (kita suruh AI jawab pake JSON)
type AIResponse struct {
	IsValid bool   `json:"is_valid"`
	Amount  int64  `json:"amount"`
	Bank    string `json:"bank"`
	Reason  string `json:"reason"`
}

func VerifyPayment(c *gin.Context) {
	// 1. Ambil User ID dari Token (Middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Ambil File Gambar dari Form Frontend
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal mengambil file gambar: " + err.Error()})
		return
	}
	defer file.Close()

	// Cek ukuran file (max 5MB biar gak berat)
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file terlalu besar (Max 5MB)"})
		return
	}

	// Baca isi file ke byte
	imgData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
		return
	}

	// 3. SIAPKAN KONEKSI KE GOOGLE GEMINI AI
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(GEMINI_API_KEY))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal koneksi ke AI: " + err.Error()})
		return
	}
	defer client.Close()

	// Pilih Model (Gemini 1.5 Flash - Cepat & Murah/Gratis)
	model := client.GenerativeModel("gemini-1.5-flash")
	
	// Set respons jadi JSON (biar gampang di-parsing)
	model.ResponseMIMEType = "application/json"

	// 4. PROMPT RAHASIA (Mantra untuk AI)
	prompt := `
		Kamu adalah asisten verifikasi pembayaran keuangan. 
		Tugasmu adalah menganalisis gambar struk/bukti transfer yang diunggah.
		
		Analisis apakah gambar ini adalah BUKTI TRANSFER YANG SAH (Struk ATM, M-Banking, atau E-Wallet).
		Cari kata kunci seperti "Berhasil", "Sukses", "Transfer", "Rp", tanggal, dan nominal.
		
		JANGAN TERIMA jika:
		1. Gambar buram/gelap total.
		2. Gambar bukan struk pembayaran (misal foto selfie, pemandangan, atau meme).
		3. Status transaksi "Gagal" atau "Pending".

		Keluarkan output JSON dengan format persis seperti ini:
		{
			"is_valid": true/false,
			"amount": 100000 (ambil angkanya saja, tanpa titik/koma),
			"bank": "Nama Bank/E-Wallet Pengirim",
			"reason": "Alasan singkat kenapa valid atau tidak valid"
		}
	`

	// Kirim Gambar + Prompt ke AI
	resp, err := model.GenerateContent(ctx,
		genai.Text(prompt),
		genai.ImageData("jpeg", imgData)) // Asumsi gambar jpeg/png, Gemini cukup pintar handle ini
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI gagal menganalisis: " + err.Error()})
		return
	}

	// 5. OLAH JAWABAN AI
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI tidak memberikan jawaban"})
		return
	}

	// Ambil teks JSON dari AI
	aiJsonText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	
	// Bersihkan format kalau AI ngasih markdown ```json ... ```
	aiJsonText = strings.TrimPrefix(aiJsonText, "```json")
	aiJsonText = strings.TrimSuffix(aiJsonText, "```")
	
	var result AIResponse
	if err := json.Unmarshal([]byte(aiJsonText), &result); err != nil {
		// Fallback kalau AI ngasih teks biasa (jarang terjadi di mode JSON)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca respon AI", "raw": aiJsonText})
		return
	}

	// 6. KEPUTUSAN FINAL
	if !result.IsValid {
		// Kalau AI bilang gak valid
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "rejected",
			"reason": result.Reason,
			"ai_analysis": result,
		})
		return
	}

	// 7. KALAU VALID -> AKTIFKAN USER!
	// 7. KALAU VALID -> AKTIFKAN USER!
    var user models.User
    if err := database.DB.First(&user, userID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
        return
    }

    // Update Status jadi ACTIVE
    user.Status = "active"
    // HAPUS BARIS INI: user.TrialEndsAt = nil 
    
    // Simpan perubahan
    database.DB.Save(&user)

    c.JSON(http.StatusOK, gin.H{
        "status": "approved",
        "message": "Pembayaran berhasil diverifikasi! Akun Anda kini AKTIF.",
        "data": gin.H{
            "user_status": "active",
            "detected_amount": result.Amount,
            "detected_bank": result.Bank,
        },
    })
}
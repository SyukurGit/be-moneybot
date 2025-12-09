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

// Gunakan API Key kamu
const GEMINI_API_KEY = "AIzaSyDzZPiPk-zTNz52eXbuAgcuPdgDhOptYU8"

type AIResponse struct {
	IsValid bool   `json:"is_valid"`
	Amount  int64  `json:"amount"`
	Bank    string `json:"bank"`
	Reason  string `json:"reason"`
}

func VerifyPayment(c *gin.Context) {
	// 1. Ambil User ID (Gunakan "user_id" sesuai middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized context"})
		return
	}

	// 2. Ambil File
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal ambil file: " + err.Error()})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File max 5MB"})
		return
	}

	imgData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal baca file"})
		return
	}

	// 3. Setup Gemini
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(GEMINI_API_KEY))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal koneksi AI: " + err.Error()})
		return
	}
	defer client.Close()

	// [PERBAIKAN] Gunakan model name yang lebih spesifik
model := client.GenerativeModel("gemini-1.5-flash")
	model.ResponseMIMEType = "application/json"

	prompt := `
		Analisis gambar bukti transfer ini.
		Output JSON valid:
		{
			"is_valid": true/false,
			"amount": 100000,
			"bank": "BCA/GOPAY/DANA",
			"reason": "singkat"
		}
		Valid jika terlihat seperti struk transfer bank/e-wallet asli dan status BERHASIL/SUKSES.
	`

	// 4. Kirim ke AI (Asumsi JPEG agar aman, Gemini auto-detect biasanya)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt), genai.ImageData("jpeg", imgData))
	if err != nil {
		// [DEBUG] Tampilkan error detail dari AI jika ada
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI Error: " + err.Error()})
		return
	}

	// 5. Parsing Jawaban
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI tidak menjawab"})
		return
	}

	aiJsonText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	aiJsonText = strings.TrimPrefix(aiJsonText, "```json")
	aiJsonText = strings.TrimSuffix(aiJsonText, "```")

	var result AIResponse
	if err := json.Unmarshal([]byte(aiJsonText), &result); err != nil {
		// Fallback manual cleaning jika JSON kotor
		aiJsonText = strings.TrimSpace(aiJsonText)
		if err2 := json.Unmarshal([]byte(aiJsonText), &result); err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal parse JSON AI", "raw": aiJsonText})
			return
		}
	}

	if !result.IsValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ditolak AI: " + result.Reason})
		return
	}

	// 6. Update User (Aktifkan)
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User hilang"})
		return
	}

	user.Status = "active"
	// user.TrialEndsAt dibiarkan saja history-nya
	
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "Akun AKTIF! Silakan refresh.",
		"data": result,
	})
}
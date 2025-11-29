package handlers

import (
	"backend-gin/database"
	"backend-gin/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type TelegramResponse struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func TelegramWebhook(c *gin.Context) {
	var payload struct {
		Message struct {
			Text string `json:"text"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	text := payload.Message.Text
	chatID := payload.Message.Chat.ID

	// --- 🔒 LOGIKA MULTI-USER DYNAMIS ---
	// Cek apakah Telegram ID ini terdaftar di tabel Users?
	var user models.User
	if err := database.DB.Where("telegram_id = ?", chatID).First(&user).Error; err != nil {
		// Jika tidak ketemu di DB -> Tolak (Silent Block)
		// User asing tidak akan bisa pakai bot ini
		c.JSON(http.StatusOK, gin.H{"status": "ignored_unregistered"})
		return
	}

	// --- LOGIKA PERINTAH (COMMANDS) ---

	// 1. FITUR DELETE - Langkah 1
	if strings.HasPrefix(text, "/del ") {
		idStr := strings.TrimPrefix(text, "/del ")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			sendReply(chatID, "⚠️ ID harus angka.")
			return
		}

		var trx models.Transaction
		// Pastikan user hanya bisa hapus data MILIKNYA SENDIRI
		if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&trx).Error; err != nil {
			sendReply(chatID, "❌ Data tidak ditemukan atau bukan milikmu.")
			return
		}

		confirmCmd := fmt.Sprintf("/yakinhapus%d", trx.ID)
		preview := fmt.Sprintf("⚠️ *KONFIRMASI HAPUS*\n\nKet: %s\nJml: %d\n\nKlik: %s", trx.Category, trx.Amount, confirmCmd)
		sendReply(chatID, preview)
		c.JSON(http.StatusOK, gin.H{"status": "confirming"})
		return
	}

	// 2. FITUR DELETE - Langkah 2
	if strings.HasPrefix(text, "/yakinhapus") {
		idStr := strings.TrimPrefix(text, "/yakinhapus")
		id, _ := strconv.Atoi(idStr)

		// Hapus hanya jika milik user ini
		result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Transaction{})

		if result.RowsAffected == 0 {
			sendReply(chatID, "⚠️ Gagal hapus (Data hilang/bukan milikmu).")
		} else {
			sendReply(chatID, "✅ Terhapus.")
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
		return
	}

	// 3. FITUR SALDO (Per User)
	if text == "/saldo" || text == "/summary" || text == "cek" {
		handleCekSaldo(chatID, user.ID) // Kirim UserID
		c.JSON(http.StatusOK, gin.H{"status": "replied"})
		return
	}

	// 4. BANTUAN
	if text == "/start" || text == "/help" {
		pesan := fmt.Sprintf("Halo %s! 🤖\nBot siap mencatat keuanganmu.\nID Terdaftar: %d", user.Username, user.TelegramID)
		sendReply(chatID, pesan)
		c.JSON(http.StatusOK, gin.H{"status": "replied"})
		return
	}

	// --- LOGIKA TRANSAKSI ---
	isTransaction := strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-")
	if !isTransaction {
		sendReply(chatID, "⚠️ Perintah tidak dikenali. Ketik /help")
		return
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		sendReply(chatID, "⚠️ Format: `+50000 Gaji`")
		return
	}

	nominalStr := parts[0]
	tipe := ""
	if strings.HasPrefix(nominalStr, "+") {
		tipe = "income"
	} else {
		tipe = "expense"
	}

	cleanNominal := strings.TrimPrefix(strings.TrimPrefix(nominalStr, "+"), "-")
	amount, _ := strconv.Atoi(cleanNominal)

	trx := models.Transaction{
		UserID:   user.ID, // PENTING: Link ke User yang sedang chat
		Amount:   amount,
		Type:     tipe,
		Category: parts[1],
		Note:     strings.Join(parts[2:], " "),
	}
	
	database.DB.Create(&trx)

	icon := "Dn"
	if tipe == "income" { icon = "UP" }
	
	pesan := fmt.Sprintf("✅ *Tersimpan!*\nID: %d\n%s Rp %d\n📂 %s", trx.ID, icon, amount, parts[1])
	sendReply(chatID, pesan)
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// Update: Terima parameter UserID untuk filter
func handleCekSaldo(chatID int64, userID uint) {
	var trx []models.Transaction
	// Hanya ambil data milik user ini
	database.DB.Where("user_id = ?", userID).Find(&trx)

	var income, expense int
	for _, t := range trx {
		if t.Type == "income" {
			income += t.Amount
		} else {
			expense += t.Amount
		}
	}
	
	sendReply(chatID, fmt.Sprintf("💰 Saldo Kamu: Rp %d\n(Masuk: %d, Keluar: %d)", income-expense, income, expense))
}

func sendReply(chatID int64, text string) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	msg := TelegramResponse{ChatID: chatID, Text: text, ParseMode: "Markdown"}
	body, _ := json.Marshal(msg)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}
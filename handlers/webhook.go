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
	"time" // PENTING: Untuk fitur daily limit

	"github.com/gin-gonic/gin"
)

// --- STRUKTUR DATA UNTUK TOMBOL TELEGRAM ---

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type TelegramResponse struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type EditMessageResponse struct {
	ChatID    int64  `json:"chat_id"`
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// -------------------------------------------

func TelegramWebhook(c *gin.Context) {
	// Payload bisa berupa Message (Chat biasa) ATAU CallbackQuery (Klik Tombol)
	var payload struct {
		Message *struct {
			Text string `json:"text"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
		CallbackQuery *struct {
			ID      string `json:"id"`
			Data    string `json:"data"`
			Message struct {
				MessageID int `json:"message_id"`
				Chat      struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
			From struct {
				ID int64 `json:"id"` // ID User yang klik tombol
			} `json:"from"`
		} `json:"callback_query"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// --- 1. HANDLING KLIK TOMBOL (CALLBACK) ---
	if payload.CallbackQuery != nil {
		chatID := payload.CallbackQuery.Message.Chat.ID
		messageID := payload.CallbackQuery.Message.MessageID
		data := payload.CallbackQuery.Data
		clickerID := payload.CallbackQuery.From.ID // ID yang klik

		// Cek Keamanan DB
		var user models.User
		if err := database.DB.Where("telegram_id = ?", clickerID).First(&user).Error; err != nil {
			return 
		}

		// LOGIKA TOMBOL HAPUS
		if strings.HasPrefix(data, "del_yes_") {
			idStr := strings.TrimPrefix(data, "del_yes_")
			id, _ := strconv.Atoi(idStr)

			// Hapus Data
			res := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Transaction{})
			
			if res.RowsAffected > 0 {
				editMessage(chatID, messageID, fmt.Sprintf("✅ *Sukses!* Data ID %d berhasil dihapus.", id))
			} else {
				editMessage(chatID, messageID, "❌ Gagal hapus. Data mungkin sudah hilang.")
			}
		} else if data == "del_cancel" {
			editMessage(chatID, messageID, "👌 Penghapusan dibatalkan.")
		
		// LOGIKA TOMBOL KATEGORI (Save Income/Expense)
		} else if strings.HasPrefix(data, "save_") {
			parts := strings.Split(data, "_")
			if len(parts) >= 4 {
				tipe := parts[1]
				amount, _ := strconv.Atoi(parts[2])
				category := parts[3]

				trx := models.Transaction{
					UserID:   user.ID,
					Amount:   amount,
					Type:     tipe,
					Category: category,
					Note:     "Via Quick Button",
				}
				database.DB.Create(&trx)

				icon := "Dn"
				alertMsg := ""

				if tipe == "income" { 
					icon = "UP" 
				} else {
					// Cek Limit Harian (Fitur Baru)
					alertMsg = checkDailyLimit(user.ID, amount)
				}
				
				finalMsg := fmt.Sprintf("✅ *Tersimpan!*\nID: %d\n%s Rp %d\n📂 %s%s", trx.ID, icon, amount, category, alertMsg)
				editMessage(chatID, messageID, finalMsg)
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "callback_processed"})
		return
	}

	// --- 2. HANDLING CHAT BIASA (MESSAGE) ---
	if payload.Message == nil {
		return 
	}

	text := payload.Message.Text
	chatID := payload.Message.Chat.ID

	// Cek User di DB
	var user models.User
	if err := database.DB.Where("telegram_id = ?", chatID).First(&user).Error; err != nil {
		pesan := "🚫 Maaf, Anda belum terdaftar dalam sistem.\n\nSilakan hubungi admin *@unxpctedd* untuk pendaftaran."
		sendReply(chatID, pesan, nil)
		c.JSON(http.StatusOK, gin.H{"status": "replied_unregistered"})
		return
	}

	// FITUR DELETE (Tampilkan Tombol)
	if strings.HasPrefix(text, "/del ") {
		idStr := strings.TrimPrefix(text, "/del ")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			sendReply(chatID, "⚠️ ID harus angka.", nil)
			return
		}

		var trx models.Transaction
		if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&trx).Error; err != nil {
			sendReply(chatID, "❌ Data tidak ditemukan.", nil)
			return
		}

		// Siapkan Keyboard
		keyboard := &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "✅ Ya, Hapus", CallbackData: fmt.Sprintf("del_yes_%d", trx.ID)},
					{Text: "❌ Batal", CallbackData: "del_cancel"},
				},
			},
		}

		msg := fmt.Sprintf("⚠️ *KONFIRMASI HAPUS*\n\nKategori: %s\nNominal: %d\n\nYakin hapus?", trx.Category, trx.Amount)
		sendReply(chatID, msg, keyboard)
		return
	}

	// FITUR LAIN (Saldo, Help)
	if text == "/saldo" || text == "/summary" || text == "cek" {
		handleCekSaldo(chatID, user.ID)
		return
	}
	if text == "/start" || text == "/help" {
		helpText :=
			"🤖 *Syukur MoneyBot*\n\n" +
			"*Perintah Utama*\n" +
			"• /saldo — Lihat total pemasukan, pengeluaran, dan saldo.\n" +
			"• /del <ID> — Hapus transaksi (dengan konfirmasi tombol).\n\n" +
			"*Input Transaksi*\n" +
			"• +50000 — Simpan pemasukan (bot meminta kategori).\n" +
			"• -20000 — Simpan pengeluaran (bot meminta kategori).\n" +
			"• +50000 Gaji — Simpan langsung dengan kategori.\n" +
			"• -20000 Makan bakso — Simpan lengkap beserta catatan.\n\n" +
			"*Tombol Otomatis*\n" +
			"• Jika input hanya angka (+ / -), bot menampilkan pilihan kategori.\n" +
			"• Saat menggunakan /del <ID>, bot menampilkan tombol *Hapus* atau *Batal*.\n\n" +
			"Jika muncul pesan *Anda belum terdaftar*, hubungi admin: @sykurr88"

		sendReply(chatID, helpText, nil)
		return
	}

	// LOGIKA TRANSAKSI PINTAR
	isTransaction := strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-")
	if !isTransaction {
		sendReply(chatID, "⚠️ Perintah tidak dikenali.", nil)
		return
	}

	parts := strings.Fields(text)
	nominalStr := parts[0]
	
	// Tentukan Tipe
	tipe := "expense"
	if strings.HasPrefix(nominalStr, "+") { tipe = "income" }

	// Bersihkan Angka
	cleanNominal := strings.TrimPrefix(strings.TrimPrefix(nominalStr, "+"), "-")
	amount, err := strconv.Atoi(cleanNominal)
	if err != nil {
		sendReply(chatID, "⚠️ Angka tidak valid.", nil)
		return
	}

	// SKENARIO 1: USER LUPA KATEGORI (Cuma ketik "+50000")
	// Tampilkan Tombol Pilihan Kategori
	if len(parts) == 1 {
		var buttons [][]InlineKeyboardButton
		
		if tipe == "income" {
			buttons = [][]InlineKeyboardButton{
				{{Text: "💰 Gaji", CallbackData: fmt.Sprintf("save_income_%d_Gaji", amount)}},
				{{Text: "🎁 Bonus", CallbackData: fmt.Sprintf("save_income_%d_Bonus", amount)}},
				{{Text: "💵 Usaha", CallbackData: fmt.Sprintf("save_income_%d_Usaha", amount)}},
			}
		} else {
			buttons = [][]InlineKeyboardButton{
				{{Text: "🍲 Makan", CallbackData: fmt.Sprintf("save_expense_%d_Makan", amount)}},
				{{Text: "🚕 Transport", CallbackData: fmt.Sprintf("save_expense_%d_Transport", amount)}},
				{{Text: "🛒 Belanja", CallbackData: fmt.Sprintf("save_expense_%d_Belanja", amount)}},
				{{Text: "⚡ Tagihan", CallbackData: fmt.Sprintf("save_expense_%d_Tagihan", amount)}},
			}
		}

		replyMarkup := &InlineKeyboardMarkup{InlineKeyboard: buttons}
		sendReply(chatID, fmt.Sprintf("📂 Pilih Kategori untuk *%s Rp %d*:", strings.ToUpper(tipe), amount), replyMarkup)
		return
	}

	// SKENARIO 2: INPUT LENGKAP ("+50000 Gaji")
	// Simpan Langsung seperti biasa
	trx := models.Transaction{
		UserID:   user.ID,
		Amount:   amount,
		Type:     tipe,
		Category: parts[1],
		Note:     strings.Join(parts[2:], " "),
	}
	database.DB.Create(&trx)

	icon := "Dn"
	alertMsg := ""

	if tipe == "income" { 
		icon = "UP" 
	} else {
		// Cek Limit Harian (Fitur Baru)
		alertMsg = checkDailyLimit(user.ID, amount)
	}
	
	pesan := fmt.Sprintf("✅ *Tersimpan!*\nID: %d\n%s Rp %d\n📂 %s%s", trx.ID, icon, amount, parts[1], alertMsg)
	sendReply(chatID, pesan, nil)
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// --- HELPERS ---

// Helper Baru: Cek apakah limit harian terlampaui?
func checkDailyLimit(userID uint, currentAmount int) string {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return ""
	}

	// Jika user tidak set limit (0), abaikan
	if user.DailyLimit <= 0 {
		return ""
	}

	// Hitung total pengeluaran HARI INI
	var totalToday int
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	database.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND type = 'expense' AND created_at >= ?", userID, startOfDay).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&totalToday)

	// Cek apakah melebih limit?
	// (Total Hari Ini SUDAH termasuk transaksi yang baru saja diinput karena checkDailyLimit dipanggil setelah Create)
	if totalToday >= user.DailyLimit {
    pesan := user.AlertMessage
    if pesan == "" {
        pesan = "🔴 Kamu sudah melebihi limit harian!"
    }

    warning := "\n\n" +
        "🚨 <i>Warning Limit : </i>\n" +
        pesan

    return warning
}

	return ""
}

func handleCekSaldo(chatID int64, userID uint) {
	var trx []models.Transaction
	database.DB.Where("user_id = ?", userID).Find(&trx)
	var inc, exp int
	for _, t := range trx {
		if t.Type == "income" { inc += t.Amount } else { exp += t.Amount }
	}
	sendReply(chatID, fmt.Sprintf("💰 Saldo: Rp %d\n(Masuk: %d, Keluar: %d)", inc-exp, inc, exp), nil)
}

// Helper Kirim Pesan (Bisa dengan Tombol / Tanpa Tombol)
func sendReply(chatID int64, text string, markup *InlineKeyboardMarkup) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	
	msg := TelegramResponse{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML", // Ubah ke HTML supaya tag <b> di alert berfungsi (atau Markdown)
		ReplyMarkup: markup,
	}
	
	body, _ := json.Marshal(msg)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

// Helper Edit Pesan (Untuk mengubah pesan tombol jadi pesan sukses)
func editMessage(chatID int64, messageID int, text string) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)

	msg := EditMessageResponse{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: "HTML", // Samakan dengan sendReply
	}

	body, _ := json.Marshal(msg)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}
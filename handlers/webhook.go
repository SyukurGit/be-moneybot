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
				ID int64 `json:"id"`
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
		clickerID := payload.CallbackQuery.From.ID

		var user models.User
		if err := database.DB.Where("telegram_id = ?", clickerID).First(&user).Error; err != nil {
			return 
		}

		if strings.HasPrefix(data, "del_yes_") {
			idStr := strings.TrimPrefix(data, "del_yes_")
			id, _ := strconv.Atoi(idStr)
			res := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Transaction{})
			
			if res.RowsAffected > 0 {
				editMessage(chatID, messageID, fmt.Sprintf("✅ *Sukses!* Data ID %d berhasil dihapus.", id))
			} else {
				editMessage(chatID, messageID, "❌ Gagal hapus. Data mungkin sudah hilang.")
			}
		} else if data == "del_cancel" {
			editMessage(chatID, messageID, "👌 Penghapusan dibatalkan.")
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
					// GUNAKAN HELPER DARI TRANSACTION.GO
					alertMsg = "\n\n🚨 " + CheckDailyLimit(user.ID, 0)
					if alertMsg == "\n\n🚨 " { alertMsg = "" } // Bersihkan jika kosong
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
		// PERUBAHAN: Menampilkan ID Telegram user secara langsung
		pesan := fmt.Sprintf("🚫 <b>Akses Ditolak</b>\n\n"+
			"Anda belum terdaftar dalam sistem ini.\n\n"+
			"👉 <b>Cara Daftar:</b>\n"+
			"1. ID Telegram kamu adalah: <code>%d</code>\n"+
			"2. Teruskan (forward) ID tersebut ke admin <b>@unxpctedd</b> untuk didaftarkan.", chatID)
		
		sendReply(chatID, pesan, nil)
		c.JSON(http.StatusOK, gin.H{"status": "replied_unregistered"})
		return
	}

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
		keyboard := &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{{Text: "✅ Ya, Hapus", CallbackData: fmt.Sprintf("del_yes_%d", trx.ID)}, {Text: "❌ Batal", CallbackData: "del_cancel"}},
			},
		}
		msg := fmt.Sprintf("⚠️ *KONFIRMASI HAPUS*\n\nKategori: %s\nNominal: %d\n\nYakin hapus?", trx.Category, trx.Amount)
		sendReply(chatID, msg, keyboard)
		return
	}

	if text == "/saldo" || text == "/summary" || text == "cek" {
		handleCekSaldo(chatID, user.ID)
		return
	}


	if text == "/start" || text == "/help" {
		helpText := `🤖 <b>DompetPintarBot</b>

<b>1. Perintah Dasar</b>
• /saldo — Cek total uang masuk, keluar, dan sisa saldo.
• /del &lt;ID&gt; — Hapus transaksi (akan muncul tombol konfirmasi).

<b>2. Cara Input di Telegram</b>
• <code>+50000</code> — Input Pemasukan (Bot akan tanya kategori).
• <code>-20000</code> — Input Pengeluaran (Bot akan tanya kategori).
• <code>+50000 Gaji</code> — Input Pemasukan Langsung.
• <code>-20000 Makan</code> — Input Pengeluaran Langsung.

<b>3. Dashboard Web (www.dompet-pintar.work.gd)</b>
• 🌐 <b>Login:</b> Buka website untuk input data, edit, dan hapus dengan lebih leluasa.
• 📊 <b>Pantau:</b> Lihat grafik analisa harian/bulanan dan download laporan Excel.

<i>Perlu bantuan, hubungi @unxpctedd</i>`

		sendReply(chatID, helpText, nil)
		return
	}

	isTransaction := strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-")
	if !isTransaction {
		sendReply(chatID, "⚠️ Perintah tidak dikenali. ketik /help", nil)
		return
	}

	parts := strings.Fields(text)
	nominalStr := parts[0]
	tipe := "expense"
	if strings.HasPrefix(nominalStr, "+") { tipe = "income" }

	cleanNominal := strings.TrimPrefix(strings.TrimPrefix(nominalStr, "+"), "-")
	amount, err := strconv.Atoi(cleanNominal)
	if err != nil {
		sendReply(chatID, "⚠️ Angka tidak valid.", nil)
		return
	}

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
		// GUNAKAN HELPER DARI TRANSACTION.GO
		alertMsg = "\n\n🚨 " + CheckDailyLimit(user.ID, 0)
		if alertMsg == "\n\n🚨 " { alertMsg = "" }
	}
	
	pesan := fmt.Sprintf("✅ *Tersimpan!*\nID: %d\n%s Rp %d\n📂 %s%s", trx.ID, icon, amount, parts[1], alertMsg)
	sendReply(chatID, pesan, nil)
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
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

func sendReply(chatID int64, text string, markup *InlineKeyboardMarkup) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	msg := TelegramResponse{ChatID: chatID, Text: text, ParseMode: "HTML", ReplyMarkup: markup}
	body, _ := json.Marshal(msg)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

func editMessage(chatID int64, messageID int, text string) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)
	msg := EditMessageResponse{ChatID: chatID, MessageID: messageID, Text: text, ParseMode: "HTML"}
	body, _ := json.Marshal(msg)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

# 💰 MoneyBot Backend (Go + Gin + Telegram)

![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![Framework](https://img.shields.io/badge/Framework-Gin-00ADD8?style=flat&logo=go)
![Database](https://img.shields.io/badge/Database-SQLite-003B57?style=flat&logo=sqlite)
![License](https://img.shields.io/badge/License-MIT-green)

**MoneyBot** adalah sistem manajemen keuangan pribadi hibrida (*Hybrid Financial Tracker*) yang menggabungkan kecepatan **Bot Telegram** dengan kelengkapan **Dashboard Web**. Dibangun dengan **Go (Golang)** untuk performa tinggi, sistem ini menangani pencatatan transaksi real-time, manajemen langganan pengguna, dan verifikasi pembayaran otomatis menggunakan **OCR (Optical Character Recognition)**.

---

## 🚀 Mengapa Project Ini Dibuat?

Sistem ini dirancang untuk menyelesaikan masalah pencatatan keuangan yang ribet.
- **Instant Input:** User bisa mencatat pengeluaran langsung dari chat Telegram (Webhooks).
- **Smart Verification:** Sistem langganan yang memverifikasi bukti transfer secara otomatis menggunakan AI/OCR tanpa perlu campur tangan admin.
- **Data Ownership:** Export laporan keuangan ke Excel kapan saja.

---

## 🌟 Fitur Unggulan (Key Features)

### 1. 🤖 Telegram Bot Integration (Webhook)
Berinteraksi langsung dengan backend melalui Telegram.
- **Smart Parsing:** Input cepat dengan format `+50000 Gaji` atau `-20000 Makan`.
- **Interactive UI:** Menggunakan *Inline Buttons* untuk kategori dan konfirmasi hapus data.
- **Real-time Feedback:** Notifikasi instan saat transaksi berhasil atau limit harian terlampaui.

### 2. 💳 Automated Payment Verification (OCR)
Fitur *Killer* untuk sistem langganan (SaaS-ready).
- **AI Powered:** Mengintegrasikan **OCR.Space API** untuk membaca teks dari gambar struk transfer.
- **Logic Validasi:** Algoritma cerdas yang mendeteksi kata kunci ("BERHASIL", "SUKSES"), nama Bank (BCA, DANA, GOPAY), dan nominal transfer.
- **Auto-Activation:** Jika valid, status user otomatis berubah menjadi `active`.

### 3. 📊 Comprehensive Dashboard API
Menyediakan endpoint RESTful untuk frontend.
- **Analitik:** Data chart harian dan breakdown per kategori.
- **Excel Export:** Generate laporan `.xlsx` menggunakan library `excelize`.
- **Budget Control:** Middleware pintar untuk membatasi transaksi jika melebihi *Daily Limit*.

### 4. 🔐 Secure & Role-Based Access
- **JWT Authentication:** Keamanan berbasis token untuk setiap request API.
- **User Roles:** Pemisahan akses antara `User` biasa dan `Super Admin`.
- **Status System:** Middleware `RequireActiveOrTrial` memastikan hanya user yang membayar/trial yang bisa akses fitur premium.

---

## 🛠️ Tech Stack & Architecture

Project ini menggunakan **Monolithic Architecture** yang efisien dan mudah di-deploy.

| Komponen | Teknologi | Deskripsi |
| :--- | :--- | :--- |
| **Language** | [Go (Golang)](https://go.dev/) | Core logic, chosen for speed & concurrency. |
| **Framework** | [Gin Gonic](https://github.com/gin-gonic/gin) | HTTP Web Framework tercepat untuk Go. |
| **Database** | [SQLite](https://www.sqlite.org/) | Database ringan, zero-configuration (via GORM). |
| **ORM** | [GORM](https://gorm.io/) | Object Relational Mapping untuk manipulasi data aman. |
| **Auth** | [JWT-Go](https://github.com/golang-jwt/jwt) | Stateless authentication mechanism. |
| **OCR** | [OCR.Space API](https://ocr.space/) | External service untuk ekstrak teks dari gambar. |
| **Report** | [Excelize](https://github.com/xuri/excelize) | Engine pembuat file Excel performa tinggi. |

### 📂 Struktur Folder
```bash
backend-gin/
├── handlers/      # Controller logic (Transaction, Payment, Webhook)
├── middleware/    # Auth (JWT) & Subscription Check
├── models/        # Database Structs (GORM)
├── utils/         # Helper functions (Token Generator)
└── main.go        # Entry point & Router Configuration

```

---

## 🔌 API Endpoints Overview

Berikut adalah beberapa endpoint utama yang tersedia:

| Method | Endpoint | Deskripsi | Auth |
| --- | --- | --- | --- |
| `POST` | `/login` | Masuk & dapatkan Token JWT | ❌ |
| `POST` | `/telegram/webhook` | Endpoint penerima pesan dari Telegram | ❌ |
| `POST` | `/api/transactions` | Input transaksi baru | ✅ |
| `GET` | `/api/chart/daily` | Data grafik keuangan harian | ✅ |
| `GET` | `/api/export` | Download laporan Excel | ✅ |
| `POST` | `/api/verify-payment` | Upload bukti bayar (Auto OCR Check) | ✅ |

---

## ⚙️ Cara Menjalankan (Installation)

### Prasyarat

* Go version 1.20+
* Git

### 1. Clone Repository

```bash
git clone [https://github.com/username-anda/be-moneybot.git](https://github.com/username-anda/be-moneybot.git)
cd be-moneybot

```

### 2. Setup Environment Variables

Buat file `.env` di root folder:

```env
JWT_SECRET=rahasia_super_aman
OCR_API_KEY=dapatkan_di_ocr_space
TELEGRAM_BOT_TOKEN=token_bot_telegram_anda
OWNER_SECRET=kunci_untuk_buat_admin

```

### 3. Install Dependencies

```bash
go mod tidy

```

### 4. Jalankan Server

```bash
go run main.go

```

Server akan berjalan di `http://localhost:8080`.

---



## 🤝 Contributing

Kontribusi sangat diterima! Silakan fork repository ini dan buat Pull Request.

1. Fork Project
2. Create Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to Branch (`git push origin feature/AmazingFeature`)
5. Open Pull Request

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

---

**Developed with ❤️ using Go.**


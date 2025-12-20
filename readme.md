# 💰 MoneyBot Backend (Go + Gin + Telegram)

![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat\&logo=go)
![Framework](https://img.shields.io/badge/Framework-Gin-00ADD8?style=flat\&logo=go)
![Database](https://img.shields.io/badge/Database-SQLite-003B57?style=flat\&logo=sqlite)
![License](https://img.shields.io/badge/License-MIT-green)

MoneyBot is a high-performance Hybrid Personal Finance Management System that combines the speed of a Telegram Bot with the completeness of a Web Dashboard. Built with Go (Golang) for efficiency and scalability, this backend handles real-time transaction recording, subscription management, and automated payment verification using OCR (Optical Character Recognition).

🔗 Live Demo: https://dompetpintar.a76labs.online

This project is designed to demonstrate production-ready backend engineering, API integration, and security best practices.

---

## 🚀 Project Motivation

Traditional expense tracking tools are often slow, manual, and fragmented. MoneyBot was created to simplify financial tracking through automation and instant interaction.

Key goals:

* **Instant Input:** Record income and expenses directly from Telegram chat using webhooks.
* **Automated Subscription Verification:** No manual admin validation—payment proof is verified automatically using OCR.
* **Data Ownership:** Users can export their financial reports to Excel anytime.

---

## 🌟 Key Features

### 1. 🤖 Telegram Bot Integration (Webhook-Based)

Seamless interaction between users and backend services.

* **Smart Parsing:** Fast input format such as `+50000 Salary` or `-20000 Lunch`.
* **Interactive UI:** Inline buttons for category selection and delete confirmations.
* **Real-Time Feedback:** Instant notifications when transactions are saved or daily limits are exceeded.

### 2. 💳 Automated Payment Verification (OCR-Powered)

A **core SaaS-ready feature** for subscription-based systems.

* **AI OCR Integration:** Uses **OCR.Space API** to extract text from payment screenshots.
* **Validation Logic:** Detects success keywords (e.g. *SUCCESS*, *COMPLETED*), supported banks/e-wallets (BCA, DANA, GoPay), and transfer amounts.
* **Auto Activation:** Valid payments automatically update user status to `active`.

### 3. 📊 Dashboard-Ready REST API

Designed to support modern frontend dashboards.

* **Analytics Endpoints:** Daily financial charts and category-based breakdowns.
* **Excel Export:** Generates `.xlsx` reports using **Excelize**.
* **Budget Control:** Smart middleware blocks transactions when daily spending limits are exceeded.

### 4. 🔐 Security & Role-Based Access Control

Built with security-first principles.

* **JWT Authentication:** Stateless, secure token-based authentication.
* **User Roles:** Separation between standard users and `Super Admin`.
* **Subscription Middleware:** `RequireActiveOrTrial` ensures only active/trial users can access premium features.

---

## 🛠️ Tech Stack & Architecture

MoneyBot uses a **Monolithic Architecture** for simplicity, reliability, and ease of deployment.

| Component          | Technology    | Description                                               |
| ------------------ | ------------- | --------------------------------------------------------- |
| **Language**       | Go (Golang)   | Core business logic with high performance and concurrency |
| **Framework**      | Gin Gonic     | Fast and minimal HTTP web framework                       |
| **Database**       | SQLite        | Lightweight, zero-configuration database                  |
| **ORM**            | GORM          | Safe and expressive database operations                   |
| **Authentication** | JWT           | Secure stateless authentication                           |
| **OCR Service**    | OCR.Space API | Automated text extraction from payment images             |
| **Reporting**      | Excelize      | High-performance Excel report generation                  |

### 📂 Project Structure

```bash
backend-gin/
├── handlers/      # Controller logic (Transactions, Payments, Telegram Webhook)
├── middleware/    # JWT auth & subscription guards
├── models/        # GORM database models
├── utils/         # Helper utilities (JWT, parsing, helpers)
└── main.go        # Application entry point & router setup
```

---

## 🔌 API Endpoints Overview

| Method | Endpoint              | Description                           | Auth |
| ------ | --------------------- | ------------------------------------- | ---- |
| `POST` | `/login`              | Authenticate and obtain JWT           | ❌    |
| `POST` | `/telegram/webhook`   | Telegram webhook receiver             | ❌    |
| `POST` | `/api/transactions`   | Create new transaction                | ✅    |
| `GET`  | `/api/chart/daily`    | Daily financial chart data            | ✅    |
| `GET`  | `/api/export`         | Download Excel financial report       | ✅    |
| `POST` | `/api/verify-payment` | Upload payment proof (OCR auto-check) | ✅    |

---

## ⚙️ Installation & Setup

### Prerequisites

* Go 1.20+
* Git

### 1. Clone Repository

```bash
git clone https://github.com/your-username/be-moneybot.git
cd be-moneybot
```

### 2. Environment Configuration

Create a `.env` file in the project root:

```env
JWT_SECRET=your_super_secure_secret
OCR_API_KEY=your_ocr_space_api_key
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
OWNER_SECRET=admin_creation_secret
```

### 3. Install Dependencies

```bash
go mod tidy
```

### 4. Run the Server

```bash
go run main.go
```

Server will start at `http://localhost:8080`.

---

## 🤝 Contributing

Contributions are welcome and appreciated.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/YourFeature`)
3. Commit your changes (`git commit -m "Add YourFeature"`)
4. Push to the branch (`git push origin feature/YourFeature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the **MIT License**. See the `LICENSE` file for details.

---

**Developed with ❤️ using Go.**

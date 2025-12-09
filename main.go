package main

import (
	"log"
	"backend-gin/database"
	"backend-gin/handlers"
	"backend-gin/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/gin-contrib/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database.ConnectDatabase()

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "ngrok-skip-browser-warning"}
	r.Use(cors.New(config))

	// Public Routes
	r.POST("/login", handlers.Login)
	r.POST("/register", handlers.Register) // Dulu register-admin, sekarang register umum
	r.POST("/telegram/webhook", handlers.TelegramWebhook)
	r.POST("/setup-owner", handlers.RegisterOwner)

	// Protected Routes (Butuh Token)
	api := r.Group("/api")
	api.Use(middleware.JwtAuthMiddleware())

    // 1. ROUTE BEBAS (Verify Payment bisa diakses walau status Suspended)
    // Diletakkan LANGSUNG di bawah 'api', sebelum middleware 'RequireActiveOrTrial'
	api.POST("/verify-payment", handlers.VerifyPayment)

    // 2. ROUTE KETAT (Butuh Token + Status Active/Trial)
    // Kita buat grup baru 'strictApi' yang menerapkan middleware tambahan
	strictApi := api.Group("/")
    strictApi.Use(middleware.RequireActiveOrTrial())
	{
		// Fitur User Biasa
		strictApi.GET("/transactions", handlers.GetTransactions)
		strictApi.GET("/summary", handlers.GetSummary)
		strictApi.GET("/chart/daily", handlers.GetDailyChart)
		strictApi.GET("/categories", handlers.GetCategorySummary)
		strictApi.GET("/user/settings", handlers.GetUserSettings)
		strictApi.PUT("/user/settings", handlers.UpdateUserSettings)
		strictApi.GET("/export", handlers.ExportExcel)

		strictApi.POST("/transactions", handlers.CreateTransaction) // Input Data
		strictApi.GET("/transactions/today", handlers.GetTodayTransactions) // Data Hari Ini
		strictApi.DELETE("/transactions/:id", handlers.DeleteTransaction) // Hapus Data

		// Fitur Super Admin (BARU)
		// Aksesnya nanti: POST /api/admin/users
		admin := strictApi.Group("/admin")
		{
			admin.GET("/users", handlers.GetAllUsers)      // Lihat semua user
			admin.POST("/users", handlers.CreateUser)      // Tambah user baru
			admin.DELETE("/users/:id", handlers.DeleteUser) // Hapus user

			admin.GET("/users/:id/stats", handlers.GetUserStats) // Get Detail
			admin.PUT("/users/:id", handlers.UpdateUser)         // Edit User
			admin.PATCH("/users/:id/status", handlers.UpdateUserStatus) // Edit Status/Trial
		}
	}

	r.Run(":8080")
}
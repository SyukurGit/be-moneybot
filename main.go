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
	api.Use(middleware.RequireActiveOrTrial()) // <-- KITA BUTUH INI
	{
		// Fitur User Biasa
		api.GET("/transactions", handlers.GetTransactions)
		api.GET("/summary", handlers.GetSummary)
		api.GET("/chart/daily", handlers.GetDailyChart)
		api.GET("/categories", handlers.GetCategorySummary)
		api.GET("/user/settings", handlers.GetUserSettings)
    api.PUT("/user/settings", handlers.UpdateUserSettings)
	api.GET("/export", handlers.ExportExcel)

	api.POST("/transactions", handlers.CreateTransaction) // Input Data
		api.GET("/transactions/today", handlers.GetTodayTransactions) // Data Hari Ini
		api.DELETE("/transactions/:id", handlers.DeleteTransaction) // Hapus Data
		api.POST("/verify-payment", handlers.VerifyPayment)

		// Fitur Super Admin (BARU)
		// Aksesnya nanti: POST /api/admin/users
		admin := api.Group("/admin")
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
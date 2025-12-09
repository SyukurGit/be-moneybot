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
	r.POST("/register", handlers.Register)
	r.POST("/telegram/webhook", handlers.TelegramWebhook)
	r.POST("/setup-owner", handlers.RegisterOwner)

	// Protected Routes (Butuh Token)
	api := r.Group("/api")
	api.Use(middleware.JwtAuthMiddleware()) 
    // PERHATIAN: JANGAN PASANG RequireActiveOrTrial DI SINI DULU!

    // 1. ROUTE BEBAS (Bisa diakses user Suspended)
    // Cuma butuh login, tidak peduli statusnya apa.
    // Verify payment harus ditaruh di sini!
	api.POST("/verify-payment", handlers.VerifyPayment) 


    // 2. ROUTE KETAT (User Suspended DILARANG MASUK)
    // Kita buat grup baru di dalam 'api' yang pakai middleware tambahan
    strictApi := api.Group("/")
	strictApi.Use(middleware.RequireActiveOrTrial()) // <--- Middleware Penjaga Pintu Dipasang DISINI
	{
		// Fitur User Biasa (User Suspended GAK BISA AKSES INI)
		strictApi.GET("/transactions", handlers.GetTransactions)
		strictApi.GET("/summary", handlers.GetSummary)
		strictApi.GET("/chart/daily", handlers.GetDailyChart)
		strictApi.GET("/categories", handlers.GetCategorySummary)
		strictApi.GET("/user/settings", handlers.GetUserSettings)
		strictApi.PUT("/user/settings", handlers.UpdateUserSettings)
		strictApi.GET("/export", handlers.ExportExcel)

		strictApi.POST("/transactions", handlers.CreateTransaction)
		strictApi.GET("/transactions/today", handlers.GetTodayTransactions)
		strictApi.DELETE("/transactions/:id", handlers.DeleteTransaction)

		// Fitur Super Admin
		admin := strictApi.Group("/admin")
		{
			admin.GET("/users", handlers.GetAllUsers)
			admin.POST("/users", handlers.CreateUser)
			admin.DELETE("/users/:id", handlers.DeleteUser)

			admin.GET("/users/:id/stats", handlers.GetUserStats)
			admin.PUT("/users/:id", handlers.UpdateUser)
			admin.PATCH("/users/:id/status", handlers.UpdateUserStatus)
		}
	}

	r.Run(":8080")
}
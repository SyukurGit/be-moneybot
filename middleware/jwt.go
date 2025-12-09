package middleware

import (
	"backend-gin/database"
	"backend-gin/models"
	"backend-gin/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Ambil Header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Butuh token akses!"})
			c.Abort()
			return
		}

		// 2. Validasi Token JWT
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return utils.ApiSecret(), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
			c.Abort()
			return
		}

		// 3. Ambil Data dari Token (Claims)
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalid"})
			c.Abort()
			return
		}

		// Ambil UserID dan Role dari token
		userIDFloat, okID := claims["user_id"].(float64)
		role, okRole := claims["role"].(string)

		if !okID || !okRole {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token corrupt"})
			c.Abort()
			return
		}
		
		userID := uint(userIDFloat)

		// Set Context untuk dipakai di Controller nanti
		c.Set("user_id", userID)
		c.Set("role", role)

		// ==========================================================
		// LOGIKA SAAS (TRIAL & SUBSCRIPTION CHECK)
		// ==========================================================

		// Admin selalu lolos (Super User)
		if role == "admin" {
			c.Next()
			return
		}

		// Ambil Data User Terbaru dari Database
		// Kita butuh status real-time, bukan status saat login
		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak ditemukan"})
			c.Abort()
			return
		}

		// Skenario 1: User sudah SUSPENDED (Masa trial habis & belum bayar)

		// if user.Status == "suspended" {
		// 	c.JSON(http.StatusPaymentRequired, gin.H{
		// 		"error": "Masa trial berakhir. Silakan lakukan pembayaran.",
		// 		"code": "PAYMENT_REQUIRED",
		// 	})
		// 	c.Abort()
		// 	return
		// }

		// Skenario 2: User masih TRIAL, tapi waktunya sudah habis
		if user.Status == "trial" && time.Now().After(user.TrialEndsAt) {
			// Update status jadi suspended secara otomatis
			user.Status = "suspended"
			database.DB.Save(&user)

			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Masa trial Anda baru saja berakhir. Akses dibekukan.",
				"code": "TRIAL_EXPIRED",
			})
			c.Abort()
			return
		}

		// Skenario 3: User ACTIVE atau TRIAL yang masih berlaku
		// Lanjut ke controller
		c.Next()
	}
}

// Middleware khusus untuk memastikan user BELUM suspended
func RequireActiveOrTrial() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, exists := c.Get("userID")
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        var user models.User
        // Pastikan import database dan models sudah benar (backend-gin/...)
        if err := database.DB.First(&user, userID).Error; err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
            return
        }

        if user.Status == "suspended" {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Masa trial berakhir. Silakan lakukan pembayaran."})
            return
        }

        c.Next()
    }
}
package middleware

import (
	"backend-gin/database"
	"backend-gin/models"
	"backend-gin/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Middleware 1: HANYA Cek Apakah Token Valid (Tanpa Cek Status)
func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Butuh token akses!"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return utils.ApiSecret(), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalid"})
			c.Abort()
			return
		}

		// FIX: Pastikan konversi float64 ke uint aman
		userIDFloat, okID := claims["user_id"].(float64)
		if !okID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token corrupt (user_id)"})
			c.Abort()
			return
		}
		
		// KONSISTENSI KEY: Gunakan "user_id" (snake_case) di seluruh aplikasi
		c.Set("user_id", uint(userIDFloat))
		c.Next()
	}
}

// Middleware 2: Penjaga Pintu Dashboard (Cek Status)
func RequireActiveOrTrial() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil pakai key yang SAMA dengan middleware di atas
		userID, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		// Skenario: User Suspended
		if user.Status == "suspended" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Masa trial berakhir. Silakan lakukan pembayaran."})
			return
		}

		c.Next()
	}
}
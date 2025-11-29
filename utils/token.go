package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Update: Menerima role juga
func GenerateToken(userID uint, role string) (string, error) {
	apiSecret := os.Getenv("JWT_SECRET") 
	
	claims := jwt.MapClaims{}
	claims["user_id"] = userID
	claims["role"] = role // BARU: Simpan jabatan di token
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(apiSecret))
}

func ApiSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}
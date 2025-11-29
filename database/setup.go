package database

import (
	"backend-gin/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	database, err := gorm.Open(sqlite.Open("finance.db"), &gorm.Config{})

	if err != nil {
		panic("Gagal konek ke database: " + err.Error())
	}

	database.AutoMigrate(&models.User{}, &models.Transaction{})

	DB = database
}
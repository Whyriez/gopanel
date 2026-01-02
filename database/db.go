package database

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique"`
	Password string
	Role     string
}

type Database struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   // Milik siapa database ini?
	DBName string `gorm:"unique"`
	DBUser string
}

type Website struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   // Milik siapa website ini?
	Domain string `gorm:"unique"`
	Type   string // static, php, proxy
	Port   string // Cuma dipakai kalau proxy (misal 3000)
}

func Connect() {
	var err error
	DB, err = gorm.Open(sqlite.Open("gopanel.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal konek ke database:", err)
	}

	DB.AutoMigrate(&User{}, &Database{}, &Website{})
	fmt.Println("✅ Database terhubung!")

	// Seed Admin
	var count int64
	DB.Model(&User{}).Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := User{
			Username: "admin",
			Password: string(hash),
			Role:     "admin",
		}
		DB.Create(&admin)
		fmt.Println("👤 User Admin dibuat! (User: admin, Pass: admin123)")
	}
}

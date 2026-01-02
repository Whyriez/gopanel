package handlers

import (
	"fmt"
	"gopanel/database"
	"gopanel/services"
	"regexp"

	"github.com/gofiber/fiber/v2"
)

type CreateDBRequest struct {
	DBName string `json:"db_name"`
	DBUser string `json:"db_user"`
	DBPass string `json:"db_pass"`
}

type DeleteDBRequest struct {
	DBName string `json:"db_name"`
	DBUser string `json:"db_user"`
}

// 1. CREATE DATABASE
func CreateDatabase(c *fiber.Ctx) error {
	// Ambil ID User yg lagi login (dari Middleware)
	userID := uint(c.Locals("user_id").(float64))

	req := new(CreateDBRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	// Validasi input kosong
	if req.DBName == "" || req.DBUser == "" || req.DBPass == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Semua field wajib diisi"})
	}

	// 1. Eksekusi di MySQL Server (Fisik)
	err := services.CreateMySQLDatabase(req.DBName, req.DBUser, req.DBPass)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 2. Simpan Catatan Kepemilikan di SQLite (Internal)
	newDB := database.Database{
		UserID: userID,
		DBName: req.DBName,
		DBUser: req.DBUser,
	}

	if err := database.DB.Create(&newDB).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database fisik dibuat, tapi gagal simpan ke panel DB"})
	}

	return c.JSON(fiber.Map{"message": "Database MySQL berhasil dibuat!"})
}

func DeleteDatabase(c *fiber.Ctx) error {
	req := new(DeleteDBRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	// Validasi nama biar aman (hanya huruf angka underscore)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validName.MatchString(req.DBName) || !validName.MatchString(req.DBUser) {
		return c.Status(400).JSON(fiber.Map{"error": "Format nama database/user tidak valid"})
	}

	db, err := services.GetDBConnection()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal konek DB"})
	}

	// Hapus Database
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", req.DBName))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal hapus database: " + err.Error()})
	}

	// Hapus User
	_, err = db.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", req.DBUser))
	// Note: Error drop user diabaikan jika user tidak ada, yg penting DB kehapus

	return c.JSON(fiber.Map{"message": "Database & User berhasil dihapus"})
}

// 2. LIST DATABASES (Milik User Sendiri)
func ListDatabases(c *fiber.Ctx) error {
	userID := uint(c.Locals("user_id").(float64))
	role := c.Locals("role").(string)

	var dbs []database.Database

	if role == "admin" {
		// Admin bisa lihat semua DB (Opsional)
		database.DB.Find(&dbs)
	} else {
		// Customer cuma bisa lihat DB miliknya
		database.DB.Where("user_id = ?", userID).Find(&dbs)
	}

	return c.JSON(dbs)
}

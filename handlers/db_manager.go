package handlers

import (
	"gopanel/database"
	"gopanel/services"

	"github.com/gofiber/fiber/v2"
)

type CreateDBRequest struct {
	DBName string `json:"db_name"`
	DBUser string `json:"db_user"`
	DBPass string `json:"db_pass"`
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

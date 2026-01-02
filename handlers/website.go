package handlers

import (
	"gopanel/database"
	"gopanel/services"
	"os/exec"

	"github.com/gofiber/fiber/v2"
)

type CreateSiteRequest struct {
	Domain   string `json:"domain"`
	Type     string `json:"type"` // static, php, proxy
	Port     string `json:"port"` // opsional (untuk node/python)
	StartCmd string `json:"start_cmd"`
}
type DeleteSiteRequest struct {
	Domain string `json:"domain"`
}

func CreateWebsite(c *fiber.Ctx) error {
	// Ambil ID user dari Token JWT (Middleware)
	userID := uint(c.Locals("user_id").(float64))

	req := new(CreateSiteRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	if req.Domain == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Domain wajib diisi"})
	}
	if req.Type == "" {
		req.Type = "static"
	}
	if req.Type == "proxy" && req.Port == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Port wajib diisi untuk proxy"})
	}

	if req.Type == "proxy" && req.StartCmd == "" {
		req.StartCmd = "node app.js"
	}

	// Cek apakah domain sudah ada (Global Check)
	var count int64
	database.DB.Model(&database.Website{}).Where("domain = ?", req.Domain).Count(&count)
	if count > 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Domain sudah terdaftar oleh user lain"})
	}

	// Generate Config Nginx
	path, err := services.GenerateNginxConfig(req.Domain, req.Type, req.Port, req.StartCmd)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Simpan ke DB
	newWeb := database.Website{
		UserID:   userID,
		Domain:   req.Domain,
		Type:     req.Type,
		Port:     req.Port,
		StartCmd: req.StartCmd, // <--- Simpan
	}

	if err := database.DB.Create(&newWeb).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database Error: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":   "Website Berhasil Dibuat!",
		"domain":    req.Domain,
		"file_path": path,
	})
}

func ListWebsites(c *fiber.Ctx) error {
	userID := uint(c.Locals("user_id").(float64))
	role := c.Locals("role").(string)

	var websites []database.Website

	if role == "admin" {
		// Admin bisa lihat SEMUA website
		database.DB.Find(&websites)
	} else {
		// Customer cuma bisa lihat website MILIK SENDIRI
		database.DB.Where("user_id = ?", userID).Find(&websites)
	}

	// Return array nama domain saja (sesuai kebutuhan frontend saat ini)
	var domains []string
	for _, w := range websites {
		domains = append(domains, w.Domain)
	}

	return c.JSON(domains)
}

func DeleteWebsite(c *fiber.Ctx) error {
	userID := uint(c.Locals("user_id").(float64))
	role := c.Locals("role").(string)

	req := new(DeleteSiteRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Format data salah"})
	}

	// Cari website di DB
	var website database.Website
	result := database.DB.Where("domain = ?", req.Domain).First(&website)
	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Website tidak ditemukan"})
	}

	// KEAMANAN: Cek apakah yang menghapus adalah pemilik asli (atau admin)
	if role != "admin" && website.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Anda tidak berhak menghapus website ini!"})
	}

	// Hapus Config Server
	if err := services.RemoveNginxConfig(website.Domain); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal hapus server config: " + err.Error()})
	}

	if website.Type == "proxy" {
		// Command: pm2 delete domain.com
		exec.Command("pm2", "delete", website.Domain).Run()
		exec.Command("pm2", "save").Run()
	}

	// Hapus Data DB
	database.DB.Delete(&website)

	return c.JSON(fiber.Map{"message": "Website berhasil dihapus!"})
}

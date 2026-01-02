package handlers

import (
	"gopanel/database"
	"gopanel/services"

	"github.com/gofiber/fiber/v2"
)

type CreateSiteRequest struct {
	Domain string `json:"domain"`
	Type   string `json:"type"` // static, php, proxy
	Port   string `json:"port"` // opsional (untuk node/python)
}
type DeleteSiteRequest struct {
	Domain string `json:"domain"`
}

func CreateWebsite(c *fiber.Ctx) error {
	req := new(CreateSiteRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	// Validasi Dasar
	if req.Domain == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Domain wajib diisi"})
	}
	if req.Type == "" {
		req.Type = "static"
	}
	if req.Type == "proxy" && req.Port == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Port wajib diisi untuk website Node/Python"})
	}

	// === 1. SIMPAN KE DATABASE DULU ===
	// (Opsional: Cek dulu apa domain sudah ada biar gak duplikat)

	// === 2. GENERATE VIA SERVICE (SERVER LOGIC) ===
	// Kita serahkan urusan bikin folder dan index.html ke Service Nginx saja.
	// Jangan bikin manual di sini biar templatenya jalan!
	path, err := services.GenerateNginxConfig(req.Domain, req.Type, req.Port)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// === 3. SIMPAN DATA KE DB ===
	// (Sesuaikan user_id dengan user yang login jika ada, sementara kita hardcode/skip dulu)
	newWeb := database.Website{
		Domain: req.Domain,
		Type:   req.Type,
		Port:   req.Port,
		// UserID: authUserId, // Nanti diisi dari token
	}
	if err := database.DB.Create(&newWeb).Error; err != nil {
		// Tampilkan pesan error asli dari database (misal: UNIQUE constraint failed)
		return c.Status(500).JSON(fiber.Map{"error": "Database Error: " + err.Error()})
	}
	return c.JSON(fiber.Map{
		"message":   "Website Berhasil Dibuat!",
		"domain":    req.Domain,
		"type":      req.Type,
		"file_path": path,
	})
}

func ListWebsites(c *fiber.Ctx) error {
	// Ambil list dari Database, BUKAN dari folder scan
	// Ini lebih akurat dan konsisten
	var websites []database.Website
	database.DB.Find(&websites)

	// Kita cuma butuh list nama domainnya aja buat frontend saat ini
	var domains []string
	for _, w := range websites {
		domains = append(domains, w.Domain)
	}

	return c.JSON(domains)
}

func DeleteWebsite(c *fiber.Ctx) error {
	// [FIX] Baca Domain dari JSON Body (karena frontend kirim JSON)
	req := new(DeleteSiteRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Format data salah"})
	}

	if req.Domain == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Domain wajib diisi"})
	}

	// 1. Cari website berdasarkan DOMAIN (Bukan ID)
	var website database.Website
	result := database.DB.Where("domain = ?", req.Domain).First(&website)

	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Website tidak ditemukan di database"})
	}

	// 2. Hapus Config Nginx & Folder
	err := services.RemoveNginxConfig(website.Domain)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal hapus config server: " + err.Error()})
	}

	// 3. Hapus dari Database
	database.DB.Delete(&website)

	return c.JSON(fiber.Map{"message": "Website " + req.Domain + " berhasil dihapus!"})
}

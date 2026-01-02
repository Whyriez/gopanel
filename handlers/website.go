package handlers

import (
	"gopanel/services"

	"github.com/gofiber/fiber/v2"
)

type CreateSiteRequest struct {
	Domain string `json:"domain"`
	Type   string `json:"type"` // static, php, proxy
	Port   string `json:"port"` // opsional (untuk node/python)
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
		req.Type = "static" // Default ke static kalau kosong
	}
	if req.Type == "proxy" && req.Port == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Port wajib diisi untuk website Node/Python"})
	}

	// Panggil Service dengan parameter baru
	path, err := services.GenerateNginxConfig(req.Domain, req.Type, req.Port)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":   "Konfigurasi Website Dibuat!",
		"domain":    req.Domain,
		"type":      req.Type,
		"file_path": path,
	})
}

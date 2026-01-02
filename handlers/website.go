package handlers

import (
	"gopanel/database"
	"gopanel/services"
	"os"
	"path/filepath"

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

	// === LOGIC BARU: BUAT FOLDER WEBSITE ===
	// Ini akan membuat folder: ./sites/namadomain.com/
	cwd, _ := os.Getwd()
	sitePath := filepath.Join(cwd, "sites", req.Domain)

	if err := os.MkdirAll(sitePath, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal membuat folder website"})
	}

	// Buat file index.html default jika tipe static agar folder tidak kosong
	if req.Type == "static" || req.Type == "php" {
		indexFile := filepath.Join(sitePath, "index.html")
		defaultContent := "<h1>Welcome to " + req.Domain + "</h1><p>Created with GoPanel</p>"
		os.WriteFile(indexFile, []byte(defaultContent), 0644)
	}
	// ========================================

	// Lanjut generate Nginx Config (arahkan root nginx ke folder baru ini)
	// Note: Anda mungkin perlu update logic services.GenerateNginxConfig
	// agar root path di config nginx mengarah ke /path/to/gopanel/sites/domain
	path, err := services.GenerateNginxConfig(req.Domain, req.Type, req.Port)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":   "Website & Folder Berhasil Dibuat!",
		"domain":    req.Domain,
		"type":      req.Type,
		"file_path": path,
	})
}

func ListWebsites(c *fiber.Ctx) error {
	cwd, _ := os.Getwd()
	sitesDir := filepath.Join(cwd, "sites")

	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		// Pastikan return array kosong explicit
		return c.JSON([]string{})
	}

	// PERBAIKAN DISINI:
	// Jangan 'var sites []string' (karena ini nil)
	// Gunakan ini:
	sites := []string{}

	for _, entry := range entries {
		if entry.IsDir() {
			sites = append(sites, entry.Name())
		}
	}

	return c.JSON(sites)
}

// Handler: Hapus Website (Folder + Config Nginx)
func DeleteWebsite(c *fiber.Ctx) error {
	// 1. Ambil ID dari parameter URL (misal: /api/website/5)
	id := c.Params("id")

	// 2. Cari dulu datanya di Database (Kita butuh Nama Domain-nya!)
	var website database.Website
	result := database.DB.First(&website, id)

	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Website tidak ditemukan"})
	}

	// === [PANGGIL DI SINI] ===
	// Hapus Config Nginx & Folder File Manager SEBELUM hapus data di DB
	// Biar kalau error, datanya gak hilang duluan
	err := services.RemoveNginxConfig(website.Domain)
	if err != nil {
		// Opsional: Tetap lanjut hapus DB walau Nginx error, atau return error.
		// Saran saya: Return error biar ketahuan kalau gagal bersih-bersih server.
		return c.Status(500).JSON(fiber.Map{"error": "Gagal hapus server config: " + err.Error()})
	}
	// =========================

	// 3. Kalau Nginx sudah bersih, baru hapus dari Database SQLite
	database.DB.Delete(&website)

	return c.JSON(fiber.Map{"message": "Website dan file server berhasil dihapus!"})
}

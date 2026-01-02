package handlers

import (
	"fmt"
	"gopanel/database"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const StorageRoot = "/var/www"

func checkAccess(userID uint, role string, requestPath string) (string, error) {
	// 1. Bersihkan path
	cleanPath := filepath.Clean(filepath.Join(StorageRoot, requestPath))

	// Pastikan tidak keluar dari Root
	if !strings.HasPrefix(cleanPath, StorageRoot) {
		return "", fmt.Errorf("akses ilegal")
	}

	// 2. Jika ADMIN, bebaskan akses
	if role == "admin" {
		return cleanPath, nil
	}

	// 3. Jika CUSTOMER (Ini Logic Isolasinya)
	// Kita ambil daftar domain milik user dari DB
	var userWebsites []database.Website
	database.DB.Where("user_id = ?", userID).Find(&userWebsites)

	// Buat map biar gampang ngecek
	allowedDomains := make(map[string]bool)
	for _, w := range userWebsites {
		allowedDomains[w.Domain] = true
	}

	// Analisa Path yang diminta:
	// Path relatif terhadap /var/www. Contoh: "domain.com/public_html"
	// Kita ambil segmen pertama (nama domainnya)
	relPath, _ := filepath.Rel(StorageRoot, cleanPath)

	if relPath == "." {
		// Kalau user minta root folder (/var/www), izinkan saja
		// TAPI nanti di fungsi ListFiles kita filter tampilannya
		return cleanPath, nil
	}

	firstSegment := strings.Split(relPath, "/")[0]

	// Cek apakah segmen pertama itu adalah domain milik user?
	if !allowedDomains[firstSegment] {
		return "", fmt.Errorf("akses ditolak: folder ini bukan milik anda")
	}

	return cleanPath, nil
}

type FileItem struct {
	Name  string `json:"name"`
	Size  string `json:"size"`
	IsDir bool   `json:"is_dir"`
}

type SaveFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func getSafePath(requestPath string) (string, error) {
	// Gabungkan root dengan request user
	fullPath := filepath.Join(StorageRoot, requestPath)

	// Bersihkan path (resolve .. dan .)
	cleanPath := filepath.Clean(fullPath)

	// Pastikan user tidak keluar dari StorageRoot
	if !strings.HasPrefix(cleanPath, StorageRoot) {
		return "", fmt.Errorf("akses ditolak: dilarang keluar dari root folder")
	}

	return cleanPath, nil
}

func ListFiles(c *fiber.Ctx) error {
	userID := uint(c.Locals("user_id").(float64))
	role := c.Locals("role").(string)
	reqPath := c.Query("path")

	// 1. Validasi Akses
	fullPath, err := checkAccess(userID, role, reqPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	// 2. Baca Folder
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return c.JSON([]interface{}{})
	}

	// 3. Filter Tampilan (Khusus Root Customer)
	// Kalau customer buka root, jangan tampilkan folder orang lain
	var filteredEntries []os.DirEntry

	if role != "admin" && (reqPath == "" || reqPath == "/") {
		// Ambil list domain saya lagi
		var userWebsites []database.Website
		database.DB.Where("user_id = ?", userID).Find(&userWebsites)
		myDomains := make(map[string]bool)
		for _, w := range userWebsites {
			myDomains[w.Domain] = true
		}

		for _, e := range entries {
			// Hanya masukkan ke list jika nama foldernya ada di database saya
			if myDomains[e.Name()] {
				filteredEntries = append(filteredEntries, e)
			}
		}
	} else {
		// Kalau admin atau bukan di root, tampilkan semua isi folder itu
		filteredEntries = entries
	}

	// 4. Format Output
	var files []fiber.Map
	for _, e := range filteredEntries {
		info, _ := e.Info()
		size := int64(0)
		if !e.IsDir() {
			size = info.Size()
		}
		files = append(files, fiber.Map{
			"name":   e.Name(),
			"is_dir": e.IsDir(),
			"size":   formatBytesFile(size),
		})
	}

	// Sorting
	sort.Slice(files, func(i, j int) bool {
		if files[i]["is_dir"].(bool) != files[j]["is_dir"].(bool) {
			return files[i]["is_dir"].(bool)
		}
		return files[i]["name"].(string) < files[j]["name"].(string)
	})

	return c.JSON(files)
}

func GetFileContent(c *fiber.Ctx) error {
	userID := uint(c.Locals("user_id").(float64))
	role := c.Locals("role").(string)

	fullPath, err := checkAccess(userID, role, c.Query("path"))
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal baca file"})
	}
	return c.JSON(fiber.Map{"content": string(content)})
}

func SaveFileContent(c *fiber.Ctx) error {
	userID := uint(c.Locals("user_id").(float64))
	role := c.Locals("role").(string)

	type SaveRequest struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	req := new(SaveRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	// 1. Validasi Akses & Simpan File
	fullPath, err := checkAccess(userID, role, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	err = os.WriteFile(fullPath, []byte(req.Content), 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal simpan: " + err.Error()})
	}

	// === [FITUR BARU] AUTO RESTART PM2 ===
	// Logic: Cek apakah file ini milik website tipe "proxy"? Kalau ya, restart PM2-nya.

	// req.Path formatnya: "domain.com/app.js" atau "domain.com/public_html/style.css"
	// Kita ambil segmen pertama yaitu nama domainnya.
	parts := strings.Split(req.Path, "/")
	if len(parts) > 0 {
		domainName := parts[0]

		// Cek tipe website di database
		var website database.Website
		// Kita cuma butuh kolom 'type' aja biar hemat
		if err := database.DB.Select("type").Where("domain = ?", domainName).First(&website).Error; err == nil {

			// Jika tipe websitenya adalah PROXY (Node.js/Python)
			if website.Type == "proxy" {
				// Jalankan restart di background (Goroutine) biar user gak nunggu loading lama
				go func(d string) {
					// Command: pm2 restart domain.com
					fmt.Println("🔄 Auto-restarting PM2 for:", d)
					exec.Command("pm2", "restart", d).Run()
				}(domainName)
			}
		}
	}
	// ======================================

	return c.JSON(fiber.Map{"message": "File berhasil disimpan & diterapkan!"})
}

// Helper untuk format ukuran file (Byte -> KB -> MB)
func formatBytesFile(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

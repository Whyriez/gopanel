package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const StorageRoot = "/var/www"

const StorageDir = "./sites"

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
	// Ambil parameter path dari URL (misal ?path=test.com/public_html)
	reqPath := c.Query("path")

	// 1. Tentukan folder target
	fullPath, err := getSafePath(reqPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	// 2. Baca isi folder
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		// Kalau folder gak ketemu, mungkin baru dibuat. Return kosong aja biar gak error merah.
		return c.JSON([]interface{}{})
	}

	// 3. Format data untuk Frontend
	var files []fiber.Map

	for _, e := range entries {
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

	// 4. Sorting: Folder di atas, File di bawah
	sort.Slice(files, func(i, j int) bool {
		// Jika tipe beda (satu folder satu file), folder menang
		if files[i]["is_dir"].(bool) != files[j]["is_dir"].(bool) {
			return files[i]["is_dir"].(bool)
		}
		// Jika tipe sama, urutkan nama a-z
		return files[i]["name"].(string) < files[j]["name"].(string)
	})

	return c.JSON(files)
}

func GetFileContent(c *fiber.Ctx) error {
	reqPath := c.Query("path")
	fullPath, err := getSafePath(reqPath)
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
	type SaveRequest struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	req := new(SaveRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	fullPath, err := getSafePath(req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	// Tulis file (Permission 0644 standard web)
	err = os.WriteFile(fullPath, []byte(req.Content), 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal simpan file: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "File berhasil disimpan!"})
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

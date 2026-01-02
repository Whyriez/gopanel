package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

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

func getSafePath(relativePath string) (string, error) {
	// Ambil absolute path dari folder project
	cwd, _ := os.Getwd()
	baseDir := filepath.Join(cwd, StorageDir)

	// Pastikan folder base ada dulu
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		os.MkdirAll(baseDir, 0755)
	}

	// Gabungkan base dengan input user
	fullPath := filepath.Join(baseDir, relativePath)
	cleanPath := filepath.Clean(fullPath)

	// Security Check: Pastikan user tidak naik ke atas (../..) keluar dari folder sites
	if !strings.HasPrefix(cleanPath, baseDir) {
		return "", fmt.Errorf("Akses Ditolak")
	}

	return cleanPath, nil
}

func ListFiles(c *fiber.Ctx) error {
	relativePath := c.Query("path", "")

	cleanPath, err := getSafePath(relativePath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		// Jika folder kosong/baru dan belum ada isinya, return kosong aja jangan error 500
		return c.JSON([]FileItem{})
	}

	var files []FileItem
	for _, entry := range entries {
		info, _ := entry.Info()
		sizeStr := "-"
		if !entry.IsDir() {
			sizeStr = fmt.Sprintf("%d B", info.Size())
		}
		files = append(files, FileItem{
			Name:  entry.Name(),
			Size:  sizeStr,
			IsDir: entry.IsDir(),
		})
	}
	return c.JSON(files)
}

func GetFileContent(c *fiber.Ctx) error {
	relativePath := c.Query("path", "")

	cleanPath, err := getSafePath(relativePath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal baca file"})
	}

	return c.JSON(fiber.Map{"path": relativePath, "content": string(content)})
}

func SaveFileContent(c *fiber.Ctx) error {
	req := new(SaveFileRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	cleanPath, err := getSafePath(req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	err = os.WriteFile(cleanPath, []byte(req.Content), 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal simpan file"})
	}

	return c.JSON(fiber.Map{"message": "File berhasil disimpan!"})
}

package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type FileItem struct {
	Name  string `json:"name"`
	Size  string `json:"size"`
	IsDir bool   `json:"is_dir"`
}

type SaveFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func ListFiles(c *fiber.Ctx) error {
	baseDir, _ := os.Getwd()
	relativePath := c.Query("path", "")
	fullPath := filepath.Join(baseDir, relativePath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{"error": "Akses Ditolak!"})
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal buka folder"})
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
	baseDir, _ := os.Getwd()
	relativePath := c.Query("path", "")
	fullPath := filepath.Join(baseDir, relativePath)

	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{"error": "Akses Ditolak!"})
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal baca file"})
	}

	return c.JSON(fiber.Map{"path": relativePath, "content": string(content)})
}

func SaveFileContent(c *fiber.Ctx) error {
	baseDir, _ := os.Getwd()
	req := new(SaveFileRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	fullPath := filepath.Join(baseDir, req.Path)
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{"error": "Akses Ditolak!"})
	}

	err := os.WriteFile(cleanPath, []byte(req.Content), 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal simpan file"})
	}

	return c.JSON(fiber.Map{"message": "File berhasil disimpan!"})
}
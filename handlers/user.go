package handlers

import (
	"gopanel/database"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func CreateUser(c *fiber.Ctx) error {
	req := new(CreateUserRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Wajib diisi"})
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := database.User{
		Username: req.Username,
		Password: string(hash),
		Role:     req.Role,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal buat user"})
	}

	return c.JSON(fiber.Map{"message": "User berhasil dibuat!"})
}

func ListUsers(c *fiber.Ctx) error {
	var users []database.User
	database.DB.Select("id, username, role").Find(&users)
	return c.JSON(users)
}
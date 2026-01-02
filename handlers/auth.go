package handlers

import (
	"gopanel/database"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var SecretKey = []byte("rahasia_negara_api_12345")

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *fiber.Ctx) error {
	req := new(LoginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Input invalid"})
	}

	var user database.User
	result := database.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Username/Password salah"})
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Username/Password salah"})
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(SecretKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal bikin token"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    t,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	})

	return c.JSON(fiber.Map{"message": "Login Berhasil!", "role": user.Role})
}

func GetMe(c *fiber.Ctx) error {
	// Mengambil data dari middleware (token JWT)
	role := c.Locals("role")
	userId := c.Locals("user_id")

	return c.JSON(fiber.Map{
		"user_id": userId,
		"role":    role,
	})
}

func Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:    "jwt",
		Expires: time.Now().Add(-(time.Hour * 2)),
	})
	return c.JSON(fiber.Map{"message": "Logout sukses"})
}

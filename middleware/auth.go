package middleware

import (
	"gopanel/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func IsAuthenticated(c *fiber.Ctx) error {
	cookie := c.Cookies("jwt")
	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return handlers.SecretKey, nil
	})

	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	claims := token.Claims.(jwt.MapClaims)
	c.Locals("user_id", claims["user_id"])
	c.Locals("role", claims["role"])

	return c.Next()
}

func RequireAdmin(c *fiber.Ctx) error {
	role := c.Locals("role")
	if role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: Khusus Admin!"})
	}
	return c.Next()
}
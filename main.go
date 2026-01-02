package main

import (
	"fmt"
	"log"

	"gopanel/database"
	"gopanel/handlers"
	"gopanel/middleware"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// 1. Konek Database
	database.Connect()

	app := fiber.New()

	// === ROUTE PUBLIC (Tanpa Login) ===
	app.Static("/login", "./public/login.html")
	app.Post("/api/login", handlers.Login)
	app.Post("/api/logout", handlers.Logout)

	// === ROUTE PROTECTED (Harus Login) ===
	api := app.Group("/api", middleware.IsAuthenticated)

	api.Get("/me", handlers.GetMe)

	// System Info
	api.Get("/system-info", handlers.GetSystemInfo)

	// Website
	api.Post("/create-website", handlers.CreateWebsite)
	api.Get("/websites", handlers.ListWebsites) // <--- BARU: List
	api.Post("/delete-website", handlers.DeleteWebsite)

	// File Manager
	api.Get("/files", handlers.ListFiles)
	api.Get("/file-content", handlers.GetFileContent)
	api.Post("/save-file", handlers.SaveFileContent)

	// === ROUTE DATABASE MANAGER ===
	api.Post("/create-database", handlers.CreateDatabase)
	api.Post("/delete-database", handlers.DeleteDatabase) // <--- BARU: Delete
	api.Get("/databases", handlers.ListDatabases)

	// === ROUTE ADMIN ONLY ===
	adminAPI := api.Group("/admin", middleware.RequireAdmin)
	adminAPI.Post("/create-user", handlers.CreateUser)
	adminAPI.Get("/users", handlers.ListUsers)

	// === DASHBOARD (Protected View) ===
	app.Use("/", func(c *fiber.Ctx) error {
		if c.Cookies("jwt") == "" {
			return c.Redirect("/login")
		}
		return c.Next()
	})
	app.Static("/", "./public")

	fmt.Println("🚀 Server jalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}

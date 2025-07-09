package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/ybke/apm/internal/routes"
)

func main() {
	// Create fiber app
	app := fiber.New(fiber.Config{
		AppName: "APM Service",
	})

	// Setup routes
	routes.SetupRoutes(app)

	// Start server
	log.Fatal(app.Listen(":3000"))
}
package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/constantine950/deploydock/config"
)

func main() {
	cfg := config.Load()

	app := fiber.New(fiber.Config{
		AppName: "DeployDock v1",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "env": cfg.AppEnv})
	})

	// Routes will be registered here as we build each module
	// Day 4: app.Post("/webhooks/git", webhook.Handler)
	// Day 8: deploy routes
	// Day 12: env var routes
	// Day 13: log streaming WebSocket

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("DeployDock server starting on :%s (env: %s)", port, cfg.AppEnv)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/constantine950/deploydock/config"
	"github.com/constantine950/deploydock/internal/webhook"
	"github.com/constantine950/deploydock/internal/worker"
)

func main() {
	cfg := config.Load()

	// Connect to Postgres
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}
	log.Println("postgres connected")

	// Connect to Redis
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to parse redis URL: %v", err)
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}
	log.Println("redis connected")

	// Fiber app
	app := fiber.New(fiber.Config{
		AppName: "DeployDock v1",
	})

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

	// Webhook routes — Day 4
	webhookHandler := webhook.NewHandler(db, rdb)
	app.Post("/webhooks/git", webhookHandler.HandlePush)

	// Build worker — Day 5 (clone + detect runtime), Day 6 (full build engine)
	pool := worker.NewPool(db, rdb)
	go pool.Start(context.Background())

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
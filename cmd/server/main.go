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
	"github.com/gofiber/websocket/v2"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/constantine950/deploydock/config"
	"github.com/constantine950/deploydock/internal/webhook"
	"github.com/constantine950/deploydock/internal/worker"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}
	log.Println("postgres connected")

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to parse redis URL: %v", err)
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}
	log.Println("redis connected")

	app := fiber.New(fiber.Config{AppName: "DeployDock v1"})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "env": cfg.AppEnv})
	})

	// Auth
	authHandler := webhook.NewAuthHandler(db)
	app.Post("/auth/register", authHandler.Register)
	app.Post("/auth/login", authHandler.Login)

	// Apps
	appsHandler := webhook.NewAppsHandler(db, rdb)
	app.Get("/apps", appsHandler.List)
	app.Post("/apps", appsHandler.Create)
	app.Get("/apps/:id", appsHandler.Get)
	app.Delete("/apps/:id", appsHandler.Delete)
	app.Post("/apps/:id/deploy", appsHandler.Deploy)

	// Webhook
	webhookHandler := webhook.NewHandler(db, rdb)
	app.Post("/webhooks/git", webhookHandler.HandlePush)

	// Env vars
	envHandler := webhook.NewEnvHandler(db)
	app.Get("/apps/:id/env", envHandler.List)
	app.Post("/apps/:id/env", envHandler.Set)
	app.Delete("/apps/:id/env/:key", envHandler.Delete)

	// Domains
	domainsHandler := webhook.NewDomainsHandler(db)
	app.Get("/apps/:id/domains", domainsHandler.List)
	app.Post("/apps/:id/domains", domainsHandler.Add)
	app.Delete("/apps/:id/domains/:domainId", domainsHandler.Remove)

	// Deployments
	deployHandler := webhook.NewDeployHandler(db)
	app.Get("/apps/:id/deployments", deployHandler.List)
	app.Get("/deployments/:id", deployHandler.Get)
	app.Post("/deployments/:id/rollback", deployHandler.Rollback)

	// Log streaming WebSocket
	logHandler := webhook.NewLogHandler(db, rdb)
	app.Get("/deployments/:id/logs", webhook.WSUpgrade, websocket.New(logHandler.Stream))

	// Build worker
	pool := worker.NewPool(db, rdb)
	go pool.Start(context.Background())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("DeployDock server starting on :%s (env: %s)", port, cfg.AppEnv)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	DockerSocket string
	Port        string
}

func Load() *Config {
	// Load .env in development — ignored if file doesn't exist
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	return &Config{
		AppEnv:       getEnv("APP_ENV", "development"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://deploydock:deploydock_secret@localhost:5432/deploydock?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "change_me_in_production"),
		DockerSocket: getEnv("DOCKER_SOCKET", "/var/run/docker.sock"),
		Port:         getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
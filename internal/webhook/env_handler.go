package webhook

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/constantine950/deploydock/pkg/crypto"
)

type EnvHandler struct {
	db *sql.DB
}

func NewEnvHandler(db *sql.DB) *EnvHandler {
	return &EnvHandler{db: db}
}

// GET /apps/:id/env — list keys only, values masked
func (h *EnvHandler) List(c *fiber.Ctx) error {
	appID := c.Params("id")

	rows, err := h.db.Query(
		"SELECT id, key, created_at FROM env_vars WHERE app_id = $1 ORDER BY key",
		appID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	type EnvVar struct {
		ID        string `json:"id"`
		Key       string `json:"key"`
		Value     string `json:"value"`
		CreatedAt string `json:"created_at"`
	}

	var vars []EnvVar
	for rows.Next() {
		var v EnvVar
		if err := rows.Scan(&v.ID, &v.Key, &v.CreatedAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scan error"})
		}
		v.Value = "***" // never expose the value
		vars = append(vars, v)
	}

	return c.JSON(fiber.Map{"env_vars": vars})
}

// POST /apps/:id/env — set env var (creates or updates)
func (h *EnvHandler) Set(c *fiber.Ctx) error {
	appID := c.Params("id")

	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Key == "" || body.Value == "" {
		return c.Status(400).JSON(fiber.Map{"error": "key and value are required"})
	}

	// Encrypt the value before storing
	encrypted, err := crypto.Encrypt(body.Value)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "encryption failed"})
	}

	id := uuid.New().String()

	_, err = h.db.Exec(`
		INSERT INTO env_vars (id, app_id, key, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (app_id, key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`, id, appID, body.Key, encrypted)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	return c.Status(201).JSON(fiber.Map{
		"key":     body.Key,
		"message": "env var set — redeploy required to apply changes",
	})
}

// DELETE /apps/:id/env/:key — delete env var
func (h *EnvHandler) Delete(c *fiber.Ctx) error {
	appID := c.Params("id")
	key := c.Params("key")

	result, err := h.db.Exec(
		"DELETE FROM env_vars WHERE app_id = $1 AND key = $2", appID, key,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "env var not found"})
	}

	return c.JSON(fiber.Map{"message": "env var deleted — redeploy required"})
}
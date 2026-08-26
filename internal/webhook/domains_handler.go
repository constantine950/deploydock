package webhook

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type DomainsHandler struct {
	db *sql.DB
}

func NewDomainsHandler(db *sql.DB) *DomainsHandler {
	return &DomainsHandler{db: db}
}

func (h *DomainsHandler) List(c *fiber.Ctx) error {
	appID := c.Params("id")

	rows, err := h.db.Query(`
		SELECT id, hostname, ssl_status, created_at
		FROM domains WHERE app_id = $1 ORDER BY created_at DESC
	`, appID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	type Domain struct {
		ID        string `json:"id"`
		Hostname  string `json:"hostname"`
		SSLStatus string `json:"ssl_status"`
		CreatedAt string `json:"created_at"`
	}

	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.Hostname, &d.SSLStatus, &d.CreatedAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scan error"})
		}
		domains = append(domains, d)
	}

	return c.JSON(fiber.Map{"domains": domains})
}

func (h *DomainsHandler) Add(c *fiber.Ctx) error {
	appID := c.Params("id")

	var body struct {
		Hostname string `json:"hostname"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Hostname == "" {
		return c.Status(400).JSON(fiber.Map{"error": "hostname is required"})
	}

	id := uuid.New().String()
	_, err := h.db.Exec(`
		INSERT INTO domains (id, app_id, hostname, ssl_status)
		VALUES ($1, $2, $3, 'pending')
	`, id, appID, body.Hostname)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to add domain: " + err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"id":         id,
		"hostname":   body.Hostname,
		"ssl_status": "pending",
		"message":    "domain added — SSL provisioning is async",
	})
}

func (h *DomainsHandler) Remove(c *fiber.Ctx) error {
	appID := c.Params("id")
	domainID := c.Params("domainId")

	result, err := h.db.Exec(
		"DELETE FROM domains WHERE id = $1 AND app_id = $2", domainID, appID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "domain not found"})
	}

	return c.JSON(fiber.Map{"message": "domain removed"})
}
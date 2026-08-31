package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type AppsHandler struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewAppsHandler(db *sql.DB, rdb *redis.Client) *AppsHandler {
	return &AppsHandler{db: db, rdb: rdb}
}

func (h *AppsHandler) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	rows, err := h.db.Query(`
		SELECT id, name, slug, repo_url, branch, runtime, status, created_at
		FROM apps WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	type App struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		RepoURL   string `json:"repo_url"`
		Branch    string `json:"branch"`
		Runtime   string `json:"runtime"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}

	var apps []App
	for rows.Next() {
		var a App
		var runtime sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &a.Slug, &a.RepoURL, &a.Branch, &runtime, &a.Status, &a.CreatedAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scan error"})
		}
		a.Runtime = runtime.String
		apps = append(apps, a)
	}
	if err := rows.Err(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "rows error"})
	}

	return c.JSON(fiber.Map{"apps": apps})
}

func (h *AppsHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var body struct {
		Name    string `json:"name"`
		RepoURL string `json:"repo_url"`
		Branch  string `json:"branch"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Name == "" || body.RepoURL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name and repo_url are required"})
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	slug := slugify(body.Name)
	id := uuid.New().String()

	_, err := h.db.Exec(`
		INSERT INTO apps (id, user_id, name, slug, repo_url, branch, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'idle')
	`, id, userID, body.Name, slug, body.RepoURL, body.Branch)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create app: " + err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"id":       id,
		"name":     body.Name,
		"slug":     slug,
		"repo_url": body.RepoURL,
		"branch":   body.Branch,
		"status":   "idle",
	})
}

func (h *AppsHandler) Get(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	appID := c.Params("id")

	var id, name, slug, repoURL, branch, status, createdAt string
	var runtime sql.NullString

	err := h.db.QueryRow(`
		SELECT id, name, slug, repo_url, branch, runtime, status, created_at
		FROM apps WHERE id = $1 AND user_id = $2
	`, appID, userID).Scan(&id, &name, &slug, &repoURL, &branch, &runtime, &status, &createdAt)

	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "app not found"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	return c.JSON(fiber.Map{
		"id":         id,
		"name":       name,
		"slug":       slug,
		"repo_url":   repoURL,
		"branch":     branch,
		"runtime":    runtime.String,
		"status":     status,
		"created_at": createdAt,
	})
}

func (h *AppsHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	appID := c.Params("id")

	// Stop any running containers for this app
	rows, err := h.db.Query(`
		SELECT id FROM deployments
		WHERE app_id = $1 AND status = 'live' AND container_id IS NOT NULL
	`, appID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var deploymentID string
			if err := rows.Scan(&deploymentID); err == nil {
				name := "deploydock-" + deploymentID[:8]
				exec.Command("docker", "stop", name).Run()
				exec.Command("docker", "rm", name).Run()
			}
		}
	}

	result, err := h.db.Exec("DELETE FROM apps WHERE id = $1 AND user_id = $2", appID, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "app not found"})
	}

	return c.JSON(fiber.Map{"message": "app deleted"})
}

func (h *AppsHandler) Deploy(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	appID := c.Params("id")

	var repoURL, branch string
	err := h.db.QueryRow(
		"SELECT repo_url, branch FROM apps WHERE id = $1 AND user_id = $2", appID, userID,
	).Scan(&repoURL, &branch)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "app not found"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	deploymentID := uuid.New().String()
	_, err = h.db.Exec(`
		INSERT INTO deployments (id, app_id, commit_sha, commit_message, status)
		VALUES ($1, $2, 'manual', 'Manual deploy', 'queued')
	`, deploymentID, appID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create deployment"})
	}

	h.db.Exec("UPDATE apps SET status = 'building', updated_at = NOW() WHERE id = $1", appID)

	type BuildJob struct {
		DeploymentID string `json:"deployment_id"`
		AppID        string `json:"app_id"`
		RepoURL      string `json:"repo_url"`
		Branch       string `json:"branch"`
		CommitSHA    string `json:"commit_sha"`
	}

	job := BuildJob{
		DeploymentID: deploymentID,
		AppID:        appID,
		RepoURL:      repoURL,
		Branch:       branch,
		CommitSHA:    "manual",
	}
	jobJSON, _ := json.Marshal(job)
	h.rdb.LPush(context.Background(), "build:queue", jobJSON)

	return c.Status(202).JSON(fiber.Map{
		"deployment_id": deploymentID,
		"status":        "queued",
	})
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var result strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}
	return strings.Trim(result.String(), "-")
}
package webhook

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"github.com/constantine950/deploydock/internal/deploy"
)

type DeployHandler struct {
	db *sql.DB
}

func NewDeployHandler(db *sql.DB) *DeployHandler {
	return &DeployHandler{db: db}
}

// GET /apps/:id/deployments — list deployments for an app
func (h *DeployHandler) List(c *fiber.Ctx) error {
	appID := c.Params("id")

	rows, err := h.db.Query(`
		SELECT id, commit_sha, commit_message, status, image_tag,
		       container_id, port, error_message, started_at, finished_at, created_at
		FROM deployments
		WHERE app_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, appID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	type Deployment struct {
		ID            string  `json:"id"`
		CommitSHA     string  `json:"commit_sha"`
		CommitMessage string  `json:"commit_message"`
		Status        string  `json:"status"`
		ImageTag      string  `json:"image_tag"`
		ContainerID   string  `json:"container_id"`
		Port          *int    `json:"port"`
		ErrorMessage  string  `json:"error_message"`
		StartedAt     string  `json:"started_at"`
		FinishedAt    string  `json:"finished_at"`
		CreatedAt     string  `json:"created_at"`
	}

	var deployments []Deployment
	for rows.Next() {
		var d Deployment
		var port sql.NullInt64
		var commitSHA, commitMsg, imageTag, containerID, errMsg, startedAt, finishedAt sql.NullString
		if err := rows.Scan(
			&d.ID, &commitSHA, &commitMsg, &d.Status, &imageTag,
			&containerID, &port, &errMsg, &startedAt, &finishedAt, &d.CreatedAt,
		); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scan error"})
		}
		d.CommitSHA = commitSHA.String
		d.CommitMessage = commitMsg.String
		d.ImageTag = imageTag.String
		d.ContainerID = containerID.String
		d.ErrorMessage = errMsg.String
		d.StartedAt = startedAt.String
		d.FinishedAt = finishedAt.String
		if port.Valid {
			p := int(port.Int64)
			d.Port = &p
		}
		deployments = append(deployments, d)
	}

	return c.JSON(fiber.Map{"deployments": deployments})
}

// GET /deployments/:id — get a single deployment
func (h *DeployHandler) Get(c *fiber.Ctx) error {
	deploymentID := c.Params("id")

	var id, appID, status, createdAt string
	var commitSHA, commitMsg, imageTag, containerID, errMsg, startedAt, finishedAt sql.NullString
	var port sql.NullInt64

	err := h.db.QueryRow(`
		SELECT id, app_id, commit_sha, commit_message, status, image_tag,
		       container_id, port, error_message, started_at, finished_at, created_at
		FROM deployments WHERE id = $1
	`, deploymentID).Scan(
		&id, &appID, &commitSHA, &commitMsg, &status, &imageTag,
		&containerID, &port, &errMsg, &startedAt, &finishedAt, &createdAt,
	)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "deployment not found"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	return c.JSON(fiber.Map{
		"id":             id,
		"app_id":         appID,
		"commit_sha":     commitSHA.String,
		"commit_message": commitMsg.String,
		"status":         status,
		"image_tag":      imageTag.String,
		"container_id":   containerID.String,
		"port":           port.Int64,
		"error_message":  errMsg.String,
		"started_at":     startedAt.String,
		"finished_at":    finishedAt.String,
		"created_at":     createdAt,
	})
}

// POST /deployments/:id/rollback
func (h *DeployHandler) Rollback(c *fiber.Ctx) error {
	deploymentID := c.Params("id")

	// Get the app ID for this deployment
	var appID string
	err := h.db.QueryRow(
		"SELECT app_id FROM deployments WHERE id = $1", deploymentID,
	).Scan(&appID)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "deployment not found"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	if err := deploy.Rollback(c.Context(), h.db, appID, deploymentID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":       "rollback successful",
		"deployment_id": deploymentID,
		"status":        "live",
	})
}
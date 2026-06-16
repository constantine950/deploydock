package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// PushEvent is the subset of a GitHub push event we care about
type PushEvent struct {
	Ref        string `json:"ref"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
	Repository struct {
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

// BuildJob is what we push onto the Redis queue
type BuildJob struct {
	DeploymentID string `json:"deployment_id"`
	AppID        string `json:"app_id"`
	RepoURL      string `json:"repo_url"`
	Branch       string `json:"branch"`
	CommitSHA    string `json:"commit_sha"`
}

type Handler struct {
	db     *sql.DB
	rdb    *redis.Client
	secret string
}

func NewHandler(db *sql.DB, rdb *redis.Client) *Handler {
	return &Handler{
		db:     db,
		rdb:    rdb,
		secret: os.Getenv("WEBHOOK_SECRET"),
	}
}

func (h *Handler) HandlePush(c *fiber.Ctx) error {
	// 1. Validate signature
	signature := c.Get("X-Hub-Signature-256")
	if h.secret != "" {
		if err := ValidateGitHubSignature(c.Body(), signature, h.secret); err != nil {
			log.Printf("webhook: invalid signature from %s", c.IP())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid signature",
			})
		}
	}

	// 2. Only handle push events
	event := c.Get("X-GitHub-Event")
	if event != "push" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "event ignored"})
	}

	// 3. Parse payload
	var payload PushEvent
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	// 4. Extract branch from ref (refs/heads/main → main)
	branch := payload.Ref
	if len(branch) > 11 {
		branch = branch[11:] // strip "refs/heads/"
	}

	log.Printf("webhook: push to %s branch %s commit %s",
		payload.Repository.FullName, branch, payload.HeadCommit.ID[:7])

	// 5. Find matching app by repo URL and branch
	var appID, appBranch string
	err := h.db.QueryRow(`
		SELECT id, branch FROM apps
		WHERE repo_url = $1 AND status != 'building' AND status != 'deploying'
		LIMIT 1
	`, payload.Repository.CloneURL).Scan(&appID, &appBranch)

	if err == sql.ErrNoRows {
		log.Printf("webhook: no app found for repo %s", payload.Repository.CloneURL)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "no matching app"})
	}
	if err != nil {
		log.Printf("webhook: db error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}

	// 6. Only deploy if push is to the app's configured branch
	if branch != appBranch {
		log.Printf("webhook: push to %s but app is on %s, ignoring", branch, appBranch)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "branch mismatch, ignored"})
	}

	// 7. Create deployment record with status: queued
	deploymentID := uuid.New().String()
	commitMsg := payload.HeadCommit.Message
	if len(commitMsg) > 255 {
		commitMsg = commitMsg[:255]
	}

	_, err = h.db.Exec(`
		INSERT INTO deployments (id, app_id, commit_sha, commit_message, status, started_at)
		VALUES ($1, $2, $3, $4, 'queued', $5)
	`, deploymentID, appID, payload.HeadCommit.ID, commitMsg, time.Now())
	if err != nil {
		log.Printf("webhook: failed to create deployment: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create deployment"})
	}

	// 8. Update app status to building
	_, err = h.db.Exec(`UPDATE apps SET status = 'building', updated_at = NOW() WHERE id = $1`, appID)
	if err != nil {
		log.Printf("webhook: failed to update app status: %v", err)
	}

	// 9. Enqueue build job in Redis
	job := BuildJob{
		DeploymentID: deploymentID,
		AppID:        appID,
		RepoURL:      payload.Repository.CloneURL,
		Branch:       branch,
		CommitSHA:    payload.HeadCommit.ID,
	}

	jobJSON, err := json.Marshal(job)
	if err != nil {
		log.Printf("webhook: failed to marshal job: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to queue job"})
	}

	ctx := context.Background()
	if err := h.rdb.LPush(ctx, "build:queue", jobJSON).Err(); err != nil {
		log.Printf("webhook: failed to enqueue job: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to queue job"})
	}

	log.Printf("webhook: deployment %s queued for app %s", deploymentID[:8], appID[:8])

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"deployment_id": deploymentID,
		"status":        "queued",
		"message":       fmt.Sprintf("deployment queued for commit %s", payload.HeadCommit.ID[:7]),
	})
}
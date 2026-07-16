package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// Rollback reactivates a previous deployment's container.
// Called by POST /deployments/:id/rollback.
func Rollback(ctx context.Context, db *sql.DB, appID, targetDeploymentID string) error {
	// 1. Get the target deployment's image tag
	var imageTag string
	var port int
	err := db.QueryRow(`
		SELECT image_tag, port FROM deployments WHERE id = $1 AND app_id = $2
	`, targetDeploymentID, appID).Scan(&imageTag, &port)
	if err == sql.ErrNoRows {
		return fmt.Errorf("deployment %s not found", targetDeploymentID)
	}
	if err != nil {
		return fmt.Errorf("db error: %w", err)
	}
	if imageTag == "" {
		return fmt.Errorf("deployment %s has no image tag", targetDeploymentID)
	}

	// 2. Stop current live container
	var currentContainerID string
	err = db.QueryRow(`
		SELECT container_id FROM deployments WHERE app_id = $1 AND status = 'live' LIMIT 1
	`, appID).Scan(&currentContainerID)
	if err == nil && currentContainerID != "" {
		log.Printf("rollback: stopping current container %s", currentContainerID)
		exec.Command("docker", "stop", "deploydock-"+currentContainerID[:8]).Run()
		exec.Command("docker", "rm", "deploydock-"+currentContainerID[:8]).Run()
		db.Exec(`UPDATE deployments SET status = 'rolled_back' WHERE container_id = $1`, currentContainerID)
	}

	// 3. Re-run the target image
	containerName := "deploydock-" + targetDeploymentID[:8]
	cmd := exec.CommandContext(ctx, "docker", "run",
		"-d",
		"--name", containerName,
		"--network", "deploydock_apps",
		"-p", fmt.Sprintf("%d:%d", port, port),
		"-e", fmt.Sprintf("PORT=%d", port),
		"--restart", "unless-stopped",
		imageTag,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	// 4. Mark as live
	_, err = db.Exec(`
		UPDATE deployments SET status = 'live', container_id = $1 WHERE id = $2
	`, containerID, targetDeploymentID)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	_, err = db.Exec(`UPDATE apps SET status = 'live' WHERE id = $1`, appID)
	if err != nil {
		return fmt.Errorf("failed to update app status: %w", err)
	}

	log.Printf("rollback: deployment %s is now live", targetDeploymentID[:8])
	return nil
}
package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// Rollback stops the current live container and re-runs a previous deployment's image.
func Rollback(ctx context.Context, db *sql.DB, appID, targetDeploymentID string) error {

	// 1. Get target deployment's image tag and port
	var imageTag string
	var port int
	err := db.QueryRowContext(ctx, `
		SELECT image_tag, port FROM deployments
		WHERE id = $1 AND app_id = $2
	`, targetDeploymentID, appID).Scan(&imageTag, &port)
	if err == sql.ErrNoRows {
		return fmt.Errorf("deployment %s not found for this app", targetDeploymentID[:8])
	}
	if err != nil {
		return fmt.Errorf("db error: %w", err)
	}
	if imageTag == "" {
		return fmt.Errorf("deployment %s has no image — cannot rollback", targetDeploymentID[:8])
	}

	log.Printf("rollback: target deployment %s image %s", targetDeploymentID[:8], imageTag)

	// 2. Find and stop the current live container
	var liveDeploymentID, liveContainerID string
	err = db.QueryRowContext(ctx, `
		SELECT id, container_id FROM deployments
		WHERE app_id = $1 AND status = 'live' AND container_id IS NOT NULL
		ORDER BY created_at DESC LIMIT 1
	`, appID).Scan(&liveDeploymentID, &liveContainerID)

	if err == nil && liveContainerID != "" {
		liveName := "deploydock-" + liveDeploymentID[:8]
		log.Printf("rollback: stopping live container %s", liveName)
		exec.Command("docker", "stop", liveName).Run()
		exec.Command("docker", "rm", liveName).Run()
		db.ExecContext(ctx, `
			UPDATE deployments SET status = 'rolled_back' WHERE id = $1
		`, liveDeploymentID)
	}

	// 3. Re-run the target image on a new port
	containerName := "deploydock-" + targetDeploymentID[:8]

	// Remove existing container with same name if it exists
	exec.Command("docker", "rm", "-f", containerName).Run()

	cmd := exec.CommandContext(ctx, "docker", "run",
		"-d",
		"--name", containerName,
		"--network", "deploydock_apps",
		"-p", fmt.Sprintf("%d:%d", port, port),
		"-e", fmt.Sprintf("PORT=%d", port),
		"--restart", "unless-stopped",
		"--label", "deploydock.app_id="+appID,
		"--label", "deploydock.deployment_id="+targetDeploymentID,
		imageTag,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start rollback container: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	log.Printf("rollback: container %s started on port %d", containerID, port)

	// 4. Health check
	if err := healthCheck(port, 30*1e9); err != nil {
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", containerName).Run()
		return fmt.Errorf("rollback health check failed: %w", err)
	}

	// 5. Mark target deployment as live
	_, err = db.ExecContext(ctx, `
		UPDATE deployments
		SET status = 'live', container_id = $1, finished_at = NOW()
		WHERE id = $2
	`, containerID, targetDeploymentID)
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// 6. Mark app as live
	db.ExecContext(ctx, `UPDATE apps SET status = 'live', updated_at = NOW() WHERE id = $1`, appID)

	log.Printf("rollback: deployment %s is now live", targetDeploymentID[:8])
	return nil
}
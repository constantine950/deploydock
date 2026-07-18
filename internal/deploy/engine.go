package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Engine struct {
	db *sql.DB
}

func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

type DeployResult struct {
	ContainerID string
	Port        int
}

// Deploy runs a zero-downtime rolling deploy:
// 1. Start new container
// 2. Health check new container
// 3. If healthy: stop old container, mark deployment live
// 4. If unhealthy: stop new container, leave old running, return error
func (e *Engine) Deploy(ctx context.Context, imageTag, appID, deploymentID string) (*DeployResult, error) {

	// 1. Find currently live deployment (if any)
	var oldContainerID string
	var oldDeploymentID string
	err := e.db.QueryRow(`
		SELECT id, container_id FROM deployments
		WHERE app_id = $1 AND status = 'live' AND container_id IS NOT NULL
		ORDER BY created_at DESC LIMIT 1
	`, appID).Scan(&oldDeploymentID, &oldContainerID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query live deployment: %w", err)
	}

	if oldContainerID != "" {
		log.Printf("[deploy %s] found live container %s — will swap after health check",
			deploymentID[:8], oldContainerID)
	}

	// 2. Pick a random port
	port := 10000 + rand.Intn(10000)

	// 3. Ensure app network exists
	exec.Command("docker", "network", "create", "deploydock_apps").Run()

	// 4. Start new container
	containerName := "deploydock-" + deploymentID[:8]
	log.Printf("[deploy %s] starting container %s on port %d", deploymentID[:8], containerName, port)

	cmd := exec.CommandContext(ctx, "docker", "run",
		"-d",
		"--name", containerName,
		"--network", "deploydock_apps",
		"-p", fmt.Sprintf("%d:%d", port, port),
		"-e", fmt.Sprintf("PORT=%d", port),
		"--restart", "unless-stopped",
		"--label", "deploydock.app_id="+appID,
		"--label", "deploydock.deployment_id="+deploymentID,
		imageTag,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}
	log.Printf("[deploy %s] new container started: %s", deploymentID[:8], containerID)

	// 5. Store container ID and port
	_, err = e.db.Exec(`
		UPDATE deployments SET container_id = $1, port = $2 WHERE id = $3
	`, containerID, port, deploymentID)
	if err != nil {
		log.Printf("[deploy %s] failed to store container info: %v", deploymentID[:8], err)
	}

	// 6. Health check new container (max 30s)
	log.Printf("[deploy %s] health checking new container on port %d...", deploymentID[:8], port)
	if err := healthCheck(port, 30*time.Second); err != nil {
		// Health check failed — stop new container, leave old running
		log.Printf("[deploy %s] health check failed, rolling back to old container", deploymentID[:8])
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", containerName).Run()
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	log.Printf("[deploy %s] new container healthy — swapping", deploymentID[:8])

	// 7. Stop old container (zero-downtime: new is already serving)
	if oldContainerID != "" {
		oldName := "deploydock-" + oldDeploymentID[:8]
		log.Printf("[deploy %s] stopping old container %s", deploymentID[:8], oldName)
		exec.Command("docker", "stop", oldName).Run()
		exec.Command("docker", "rm", oldName).Run()

		// Mark old deployment as superseded
		e.db.Exec(`
			UPDATE deployments SET status = 'rolled_back' WHERE id = $1
		`, oldDeploymentID)
	}

	log.Printf("[deploy %s] container live on port %d", deploymentID[:8], port)

	return &DeployResult{
		ContainerID: containerID,
		Port:        port,
	}, nil
}

// healthCheck polls http://localhost:<port> until a response or timeout.
func healthCheck(port int, timeout time.Duration) error {
	url := fmt.Sprintf("http://localhost:%d", port)
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("container did not become healthy within %s", timeout)
}

// StopContainer stops and removes a container by deployment ID prefix.
func StopContainer(deploymentID string) {
	name := "deploydock-" + deploymentID[:8]
	exec.Command("docker", "stop", name).Run()
	exec.Command("docker", "rm", name).Run()
}
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

// Deploy runs a container from the built image, assigns a port,
// connects it to the app network, and health checks it.
func (e *Engine) Deploy(ctx context.Context, imageTag, appID, deploymentID string) (*DeployResult, error) {

	// 1. Pick a random available port in range 10000-20000
	port := 10000 + rand.Intn(10000)

	// 2. Ensure the deploydock network exists
	exec.Command("docker", "network", "create", "deploydock_apps").Run()

	// 3. Run the container
	log.Printf("[deploy %s] starting container from %s on port %d", deploymentID[:8], imageTag, port)

	containerName := "deploydock-" + deploymentID[:8]

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
	log.Printf("[deploy %s] container started: %s", deploymentID[:8], containerID)

	// 4. Store container ID and port on deployment
	_, err = e.db.Exec(`
		UPDATE deployments SET container_id = $1, port = $2 WHERE id = $3
	`, containerID, port, deploymentID)
	if err != nil {
		log.Printf("[deploy %s] failed to store container info: %v", deploymentID[:8], err)
	}

	// 5. Health check — poll until ready or timeout
	log.Printf("[deploy %s] health checking on port %d...", deploymentID[:8], port)
	if err := e.healthCheck(port, 30*time.Second); err != nil {
		// Stop the failed container
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", containerName).Run()
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	log.Printf("[deploy %s] container healthy on port %d", deploymentID[:8], port)

	return &DeployResult{
		ContainerID: containerID,
		Port:        port,
	}, nil
}

// healthCheck polls http://localhost:<port> until it gets a response or times out.
func (e *Engine) healthCheck(port int, timeout time.Duration) error {
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

// Stop stops and removes a container by name.
func Stop(containerID string) error {
	name := "deploydock-" + containerID[:8]
	exec.Command("docker", "stop", name).Run()
	exec.Command("docker", "rm", name).Run()
	return nil
}
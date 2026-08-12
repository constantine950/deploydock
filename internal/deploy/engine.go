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

// Deploy runs a zero-downtime rolling deploy without env vars.
func (e *Engine) Deploy(ctx context.Context, imageTag, appID, deploymentID string) (*DeployResult, error) {
	return e.DeployWithEnv(ctx, imageTag, appID, deploymentID, nil)
}

// DeployWithEnv runs a zero-downtime rolling deploy with env vars injected.
func (e *Engine) DeployWithEnv(ctx context.Context, imageTag, appID, deploymentID string, envVars map[string]string) (*DeployResult, error) {

	// 1. Find currently live deployment
	var oldContainerID, oldDeploymentID string
	e.db.QueryRow(`
		SELECT id, container_id FROM deployments
		WHERE app_id = $1 AND status = 'live' AND container_id IS NOT NULL
		ORDER BY created_at DESC LIMIT 1
	`, appID).Scan(&oldDeploymentID, &oldContainerID)

	if oldContainerID != "" {
		log.Printf("[deploy %s] found live container %s — will swap after health check",
			deploymentID[:8], oldContainerID)
	}

	// 2. Pick a random port
	port := 10000 + rand.Intn(10000)

	// 3. Ensure app network exists
	exec.Command("docker", "network", "create", "deploydock_apps").Run()

	// 4. Build docker run args
	containerName := "deploydock-" + deploymentID[:8]
	args := []string{
		"run", "-d",
		"--name", containerName,
		"--network", "deploydock_apps",
		"-p", fmt.Sprintf("%d:%d", port, port),
		"-e", fmt.Sprintf("PORT=%d", port),
		"--restart", "unless-stopped",
		"--label", "deploydock.app_id=" + appID,
		"--label", "deploydock.deployment_id=" + deploymentID,
	}

	// 5. Inject env vars
	for k, v := range envVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, imageTag)

	log.Printf("[deploy %s] starting container %s on port %d with %d env vars",
		deploymentID[:8], containerName, port, len(envVars))

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}
	log.Printf("[deploy %s] container started: %s", deploymentID[:8], containerID)

	// 6. Store container ID and port
	e.db.Exec(`UPDATE deployments SET container_id = $1, port = $2 WHERE id = $3`,
		containerID, port, deploymentID)

	// 7. Health check new container
	log.Printf("[deploy %s] health checking on port %d...", deploymentID[:8], port)
	if err := healthCheck(port, 30*time.Second); err != nil {
		log.Printf("[deploy %s] health check failed — rolling back", deploymentID[:8])
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", containerName).Run()
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	log.Printf("[deploy %s] new container healthy — swapping", deploymentID[:8])

	// 8. Stop old container
	if oldContainerID != "" {
		oldName := "deploydock-" + oldDeploymentID[:8]
		log.Printf("[deploy %s] stopping old container %s", deploymentID[:8], oldName)
		exec.Command("docker", "stop", oldName).Run()
		exec.Command("docker", "rm", oldName).Run()
		e.db.Exec(`UPDATE deployments SET status = 'rolled_back' WHERE id = $1`, oldDeploymentID)
	}

	log.Printf("[deploy %s] live on port %d", deploymentID[:8], port)

	return &DeployResult{
		ContainerID: containerID,
		Port:        port,
	}, nil
}

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

func StopContainer(deploymentID string) {
	name := "deploydock-" + deploymentID[:8]
	exec.Command("docker", "stop", name).Run()
	exec.Command("docker", "rm", name).Run()
}
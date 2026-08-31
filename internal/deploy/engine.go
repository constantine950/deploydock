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

func (e *Engine) Deploy(ctx context.Context, imageTag, appID, deploymentID string) (*DeployResult, error) {
	return e.DeployWithEnv(ctx, imageTag, appID, deploymentID, nil)
}

func (e *Engine) DeployWithEnv(ctx context.Context, imageTag, appID, deploymentID string, envVars map[string]string) (*DeployResult, error) {

	var oldContainerID, oldDeploymentID string
	e.db.QueryRow(`
		SELECT id, container_id FROM deployments
		WHERE app_id = $1 AND status = 'live' AND container_id IS NOT NULL
		ORDER BY created_at DESC LIMIT 1
	`, appID).Scan(&oldDeploymentID, &oldContainerID)

	port := 10000 + rand.Intn(10000)

	exec.Command("docker", "network", "create", "deploydock_apps").Run()

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

	e.db.Exec(`UPDATE deployments SET container_id = $1, port = $2 WHERE id = $3`,
		containerID, port, deploymentID)

	log.Printf("[deploy %s] health checking container %s...", deploymentID[:8], containerName)
	if err := healthCheckContainer(containerName, port, 30*time.Second); err != nil {
		log.Printf("[deploy %s] health check failed — rolling back", deploymentID[:8])
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", containerName).Run()
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	log.Printf("[deploy %s] container healthy — swapping", deploymentID[:8])

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

func healthCheckContainer(containerName string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cmd := exec.Command("docker", "exec", containerName,
			"wget", "-q", "-O", "/dev/null",
			fmt.Sprintf("http://localhost:%d", port),
		)
		if err := cmd.Run(); err == nil {
			return nil
		}

		cmd2 := exec.Command("docker", "exec", containerName,
			"sh", "-c",
			fmt.Sprintf("curl -sf http://localhost:%d || wget -q -O /dev/null http://localhost:%d", port, port),
		)
		if err := cmd2.Run(); err == nil {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("container did not become healthy within %s", timeout)
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
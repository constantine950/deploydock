package build

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BuildResult struct {
	ImageTag string
	Runtime  Runtime
}

type Engine struct {
	db *sql.DB
}

func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// Build detects runtime, writes Dockerfile, builds image via docker CLI, streams logs.
func (e *Engine) Build(ctx context.Context, repoPath, appID, deploymentID string, logger *Logger) (*BuildResult, error) {

	// 1. Detect runtime
	runtime, err := DetectRuntime(repoPath)
	if err != nil {
		return nil, fmt.Errorf("runtime detection failed: %w", err)
	}
	logger.Stdout("detected runtime: " + string(runtime))

	// 2. Write Dockerfile
	dockerfile, err := DockerfileTemplate(runtime)
	if err != nil {
		return nil, fmt.Errorf("no dockerfile template: %w", err)
	}
	dockerfilePath := filepath.Join(repoPath, "Dockerfile.deploydock")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return nil, fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	defer os.Remove(dockerfilePath)
	logger.Stdout("selected Dockerfile template for " + string(runtime))

	// 3. Build image tag
	imageTag := "deploydock/" + appID + ":" + deploymentID
	logger.Stdout("building image " + imageTag)

	// 4. Shell out to docker build — streams output in real time
	cmd := exec.CommandContext(ctx, "docker", "build",
		"-f", dockerfilePath,
		"-t", imageTag,
		"--progress=plain",
		repoPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start docker build: %w", err)
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r\n")
			if line != "" {
				logger.Stdout(line)
			}
		}
	}()

	// Stream stderr (docker build writes progress to stderr)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r\n")
			if line != "" {
				logger.Stdout(line) // docker build progress is not really an error
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("docker build failed: %w", err)
	}

	// 5. Record image tag on deployment
	_, err = e.db.Exec("UPDATE deployments SET image_tag = $1 WHERE id = $2", imageTag, deploymentID)
	if err != nil {
		logger.Stderr("failed to record image tag: " + err.Error())
	}

	logger.Stdout("image built successfully: " + imageTag)

	return &BuildResult{
		ImageTag: imageTag,
		Runtime:  runtime,
	}, nil
}
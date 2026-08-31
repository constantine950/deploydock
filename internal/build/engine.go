package build

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
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

func (e *Engine) Build(ctx context.Context, repoPath, appID, deploymentID string, logger *Logger) (*BuildResult, error) {

	// 1. Detect runtime
	runtime, err := DetectRuntime(repoPath)
	if err != nil {
		return nil, fmt.Errorf("runtime detection failed: %w", err)
	}
	logger.Stdout("detected runtime: " + string(runtime))

	// 2. Validate the repo is runnable
	if err := ValidateRunnable(repoPath, runtime); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	logger.Stdout("validation passed: repo has a runnable entry point")

	// 3. Write Dockerfile
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

	// 4. Build image
	imageTag := "deploydock/" + appID + ":" + deploymentID
	logger.Stdout("building image " + imageTag)

	cmd := exec.CommandContext(ctx, "docker", "build",
		"-f", dockerfilePath,
		"-t", imageTag,
		repoPath,
	)
	cmd.Env = append(cmd.Environ(), "DOCKER_BUILDKIT=0")

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

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r\n")
			if line != "" {
				logger.Stdout(line)
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r\n")
			var msg struct {
				Stream string `json:"stream"`
				Error  string `json:"error"`
			}
			if err := json.Unmarshal([]byte(line), &msg); err == nil {
				if msg.Error != "" {
					logger.Stderr(msg.Error)
					return
				}
				if msg.Stream != "" {
					out := strings.TrimRight(msg.Stream, "\n")
					if out != "" {
						logger.Stdout(out)
					}
					return
				}
			}
			if line != "" {
				logger.Stdout(line)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("docker build failed: %w", err)
	}

	e.db.Exec("UPDATE deployments SET image_tag = $1 WHERE id = $2", imageTag, deploymentID)
	logger.Stdout("image built successfully: " + imageTag)

	return &BuildResult{
		ImageTag: imageTag,
		Runtime:  runtime,
	}, nil
}
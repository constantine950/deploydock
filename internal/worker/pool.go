package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/redis/go-redis/v9"

	"github.com/constantine950/deploydock/internal/build"
	"github.com/constantine950/deploydock/internal/deploy"
)

type BuildJob struct {
	DeploymentID string `json:"deployment_id"`
	AppID        string `json:"app_id"`
	RepoURL      string `json:"repo_url"`
	Branch       string `json:"branch"`
	CommitSHA    string `json:"commit_sha"`
}

type Pool struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewPool(db *sql.DB, rdb *redis.Client) *Pool {
	return &Pool{db: db, rdb: rdb}
}

func (p *Pool) Start(ctx context.Context) {
	log.Println("build worker: started, watching build:queue")

	for {
		select {
		case <-ctx.Done():
			log.Println("build worker: shutting down")
			return
		default:
		}

		result, err := p.rdb.BRPop(ctx, 5_000_000_000, "build:queue").Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			log.Printf("build worker: redis error: %v", err)
			continue
		}

		var job BuildJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			log.Printf("build worker: failed to parse job: %v", err)
			continue
		}

		p.processJob(job)
	}
}

func (p *Pool) processJob(job BuildJob) {
	logger := build.NewLogger(p.db, job.DeploymentID)
	logger.Stdout("starting build for deployment " + job.DeploymentID)

	_, err := p.db.Exec("UPDATE deployments SET status = 'building' WHERE id = $1", job.DeploymentID)
	if err != nil {
		logger.Stderr("failed to update deployment status: " + err.Error())
		return
	}

	// 1. Clone repo
	tmpDir, err := os.MkdirTemp("", "deploydock-build-*")
	if err != nil {
		p.failDeployment(job, logger, "failed to create temp dir: "+err.Error())
		return
	}
	defer os.RemoveAll(tmpDir)

	logger.Stdout("cloning " + job.RepoURL + " (branch " + job.Branch + ") into " + filepath.Base(tmpDir))
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", job.Branch, job.RepoURL, tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.failDeployment(job, logger, "git clone failed: "+string(output))
		return
	}
	logger.Stdout("clone complete")

	// 2. Build image
	engine := build.NewEngine(p.db)
	buildResult, err := engine.Build(context.Background(), tmpDir, job.AppID, job.DeploymentID, logger)
	if err != nil {
		p.failDeployment(job, logger, "build failed: "+err.Error())
		return
	}

	// 3. Update app runtime
	_, err = p.db.Exec(
		"UPDATE apps SET runtime = $1, updated_at = NOW() WHERE id = $2",
		string(buildResult.Runtime), job.AppID,
	)
	if err != nil {
		logger.Stderr("failed to update app runtime: " + err.Error())
	}

	// 4. Deploy container
	logger.Stdout("starting deploy engine...")
	_, err = p.db.Exec("UPDATE deployments SET status = 'deploying' WHERE id = $1", job.DeploymentID)
	if err != nil {
		logger.Stderr("failed to update deployment status: " + err.Error())
	}

	deployEngine := deploy.NewEngine(p.db)
	deployResult, err := deployEngine.Deploy(
		context.Background(),
		buildResult.ImageTag,
		job.AppID,
		job.DeploymentID,
	)
	if err != nil {
		p.failDeployment(job, logger, "deploy failed: "+err.Error())
		return
	}

	// 5. Mark deployment live
	_, err = p.db.Exec(
		"UPDATE deployments SET status = 'live', finished_at = NOW() WHERE id = $1",
		job.DeploymentID,
	)
	if err != nil {
		logger.Stderr("failed to mark deployment live: " + err.Error())
	}

	// 6. Mark app live
	_, err = p.db.Exec("UPDATE apps SET status = 'live', updated_at = NOW() WHERE id = $1", job.AppID)
	if err != nil {
		logger.Stderr("failed to mark app live: " + err.Error())
	}

	logger.Stdout(
		"deployment live — container " + deployResult.ContainerID +
			" running on port " + fmt.Sprint(deployResult.Port),
	)
}

func (p *Pool) failDeployment(job BuildJob, logger *build.Logger, errMsg string) {
	logger.Stderr(errMsg)
	p.db.Exec(
		"UPDATE deployments SET status = 'failed', error_message = $1, finished_at = NOW() WHERE id = $2",
		errMsg, job.DeploymentID,
	)
	p.db.Exec("UPDATE apps SET status = 'idle' WHERE id = $1", job.AppID)
}
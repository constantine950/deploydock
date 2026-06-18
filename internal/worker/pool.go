package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/redis/go-redis/v9"

	"github.com/constantine950/deploydock/internal/build"
)

// BuildJob mirrors webhook.BuildJob — duplicated here to avoid an import
// cycle between webhook and worker packages.
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

// Start runs the build worker loop in the current goroutine.
// Call with `go pool.Start(ctx)` from main.
func (p *Pool) Start(ctx context.Context) {
	log.Println("build worker: started, watching build:queue")

	for {
		select {
		case <-ctx.Done():
			log.Println("build worker: shutting down")
			return
		default:
		}

		// BRPop blocks for up to 5s waiting for a job
		result, err := p.rdb.BRPop(ctx, 5_000_000_000, "build:queue").Result()
		if err == redis.Nil {
			continue // no job, loop again
		}
		if err != nil {
			log.Printf("build worker: redis error: %v", err)
			continue
		}

		// result[0] is the key name, result[1] is the value
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

	// Update status to building
	_, err := p.db.Exec(`UPDATE deployments SET status = 'building' WHERE id = $1`, job.DeploymentID)
	if err != nil {
		logger.Stderr("failed to update deployment status: " + err.Error())
		return
	}

	// 1. Clone repo to temp directory
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

	// 2. Detect runtime
	runtime, err := build.DetectRuntime(tmpDir)
	if err != nil {
		p.failDeployment(job, logger, "runtime detection failed: "+err.Error())
		return
	}
	logger.Stdout("detected runtime: " + string(runtime))

	// 3. Select Dockerfile template (validated here; written to disk in Day 6's build engine)
	if _, err := build.DockerfileTemplate(runtime); err != nil {
		p.failDeployment(job, logger, "template selection failed: "+err.Error())
		return
	}
	logger.Stdout("selected Dockerfile template for " + string(runtime))

	// 4. Record detected runtime on the app
	_, err = p.db.Exec(`UPDATE apps SET runtime = $1, status = 'idle' WHERE id = $2`, string(runtime), job.AppID)
	if err != nil {
		logger.Stderr("failed to record runtime on app: " + err.Error())
	}

	logger.Stdout("Day 5 complete — runtime detection done. Build engine (Day 6) picks up from here.")
}

func (p *Pool) failDeployment(job BuildJob, logger *build.Logger, errMsg string) {
	logger.Stderr(errMsg)

	_, err := p.db.Exec(`
		UPDATE deployments SET status = 'failed', error_message = $1, finished_at = NOW()
		WHERE id = $2
	`, errMsg, job.DeploymentID)
	if err != nil {
		log.Printf("build worker: failed to mark deployment failed: %v", err)
	}

	// Reset app status so the next push isn't blocked by a stuck "building" state
	_, err = p.db.Exec(`UPDATE apps SET status = 'idle' WHERE id = $1`, job.AppID)
	if err != nil {
		log.Printf("build worker: failed to reset app status: %v", err)
	}
}
package build

import (
	"context"
	"database/sql"
	"log"

	"github.com/redis/go-redis/v9"
)

type Logger struct {
	db           *sql.DB
	rdb          *redis.Client
	deploymentID string
}

func NewLogger(db *sql.DB, deploymentID string) *Logger {
	return &Logger{db: db, deploymentID: deploymentID}
}

// NewLoggerWithRedis creates a logger that also streams to Redis pub/sub.
func NewLoggerWithRedis(db *sql.DB, rdb *redis.Client, deploymentID string) *Logger {
	return &Logger{db: db, rdb: rdb, deploymentID: deploymentID}
}

func (l *Logger) Stdout(line string) {
	l.write("stdout", line)
}

func (l *Logger) Stderr(line string) {
	l.write("stderr", line)
}

func (l *Logger) write(stream, line string) {
	log.Printf("[deploy %s] %s", l.deploymentID[:8], line)

	// Write to build_logs table
	_, err := l.db.Exec(`
		INSERT INTO build_logs (deployment_id, stream, line)
		VALUES ($1, $2, $3)
	`, l.deploymentID, stream, line)
	if err != nil {
		log.Printf("[deploy %s] failed to write log: %v", l.deploymentID[:8], err)
	}

	// Publish to Redis pub/sub for live streaming
	if l.rdb != nil {
		channel := "logs:" + l.deploymentID
		l.rdb.Publish(context.Background(), channel, stream+"|"+line)
	}
}
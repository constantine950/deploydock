package build

import (
	"database/sql"
	"log"
)

type Logger struct {
	db           *sql.DB
	deploymentID string
}

func NewLogger(db *sql.DB, deploymentID string) *Logger {
	return &Logger{db: db, deploymentID: deploymentID}
}

// Stdout writes a stdout line to build_logs and stdout for visibility
func (l *Logger) Stdout(line string) {
	l.write("stdout", line)
}

// Stderr writes a stderr line to build_logs and stdout for visibility
func (l *Logger) Stderr(line string) {
	l.write("stderr", line)
}

func (l *Logger) write(stream, line string) {
	log.Printf("[deploy %s] %s", l.deploymentID[:8], line)

	_, err := l.db.Exec(`
		INSERT INTO build_logs (deployment_id, stream, line)
		VALUES ($1, $2, $3)
	`, l.deploymentID, stream, line)
	if err != nil {
		log.Printf("[deploy %s] failed to write log: %v", l.deploymentID[:8], err)
	}
}
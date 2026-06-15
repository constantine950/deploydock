package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://deploydock:deploydock_secret@localhost:5432/deploydock?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Create migrations tracking table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("failed to create migrations table: %v", err)
	}

	// Read migration files
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		log.Fatalf("failed to read migrations: %v", err)
	}
	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)

		// Skip if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", filename).Scan(&count)
		if err != nil {
			log.Fatalf("failed to check migration %s: %v", filename, err)
		}
		if count > 0 {
			fmt.Printf("  skip  %s (already applied)\n", filename)
			continue
		}

		// Read and execute
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read %s: %v", filename, err)
		}

		// Skip seed files
		if strings.Contains(filename, "seed") {
			continue
		}

		_, err = db.Exec(string(content))
		if err != nil {
			log.Fatalf("failed to apply %s: %v\n%s", filename, err, content)
		}

		_, err = db.Exec("INSERT INTO schema_migrations (filename) VALUES ($1)", filename)
		if err != nil {
			log.Fatalf("failed to record migration %s: %v", filename, err)
		}

		fmt.Printf("  apply  %s\n", filename)
	}

	fmt.Println("Migrations complete.")
}
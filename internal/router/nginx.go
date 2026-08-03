package router

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const nginxConfigPath = "/etc/nginx/conf.d/deploydock.conf"
const defaultDomain = "deploydock.local"

type Manager struct {
	db         *sql.DB
	configPath string
	domain     string
}

func NewManager(db *sql.DB) *Manager {
	domain := os.Getenv("DEPLOYDOCK_DOMAIN")
	if domain == "" {
		domain = defaultDomain
	}
	return &Manager{
		db:         db,
		configPath: nginxConfigPath,
		domain:     domain,
	}
}

func (m *Manager) Sync(ctx context.Context) error {
	routes, err := m.loadRoutes(ctx)
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}

	config := GenerateNginxConfig(routes)

	if err := os.WriteFile(m.configPath, []byte(config), 0644); err != nil {
		// In dev nginx may not be installed — log and continue
		log.Printf("nginx: could not write config: %v", err)
		return nil
	}

	if err := m.reload(); err != nil {
		return fmt.Errorf("nginx reload failed: %w", err)
	}

	log.Printf("nginx: synced %d routes", len(routes))
	return nil
}

func (m *Manager) loadRoutes(ctx context.Context) ([]AppRoute, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT a.slug, d.port, COALESCE(
			(SELECT string_agg(hostname, ',') FROM domains WHERE app_id = a.id AND ssl_status = 'active'),
			''
		) as custom_domains
		FROM apps a
		JOIN deployments d ON d.app_id = a.id AND d.status = 'live'
		WHERE a.status = 'live' AND d.port IS NOT NULL
		ORDER BY a.slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []AppRoute
	for rows.Next() {
		var r AppRoute
		var customDomainsStr string
		if err := rows.Scan(&r.Slug, &r.ContainerPort, &customDomainsStr); err != nil {
			return nil, err
		}
		r.Domain = m.domain
		if customDomainsStr != "" {
			for _, d := range strings.Split(customDomainsStr, ",") {
				if d != "" {
					r.CustomDomains = append(r.CustomDomains, d)
				}
			}
		}
		routes = append(routes, r)
	}

	return routes, rows.Err()
}

func (m *Manager) reload() error {
	cmd := exec.Command("nginx", "-s", "reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("nginx reload: %v — %s", err, string(output))
		log.Println("nginx: skipping reload in dev (nginx not running)")
		return nil
	}
	return nil
}
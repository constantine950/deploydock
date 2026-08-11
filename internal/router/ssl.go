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

type SSLManager struct {
	db     *sql.DB
	email  string
	webroot string
}

func NewSSLManager(db *sql.DB) *SSLManager {
	return &SSLManager{
		db:      db,
		email:   os.Getenv("LETSENCRYPT_EMAIL"),
		webroot: "/var/www/certbot",
	}
}

// ProvisionCert requests a Let's Encrypt cert for a domain via certbot.
// Updates ssl_status on the domain record to 'active' or 'failed'.
func (s *SSLManager) ProvisionCert(ctx context.Context, domainID, hostname string) error {
	if s.email == "" {
		return fmt.Errorf("LETSENCRYPT_EMAIL not set — cannot provision cert")
	}

	log.Printf("ssl: requesting cert for %s", hostname)

	// Ensure webroot exists
	os.MkdirAll(s.webroot, 0755)

	cmd := exec.CommandContext(ctx, "certbot", "certonly",
		"--webroot",
		"--webroot-path", s.webroot,
		"--email", s.email,
		"--agree-tos",
		"--no-eff-email",
		"--non-interactive",
		"-d", hostname,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ssl: certbot failed for %s: %s", hostname, string(output))
		s.db.Exec("UPDATE domains SET ssl_status = 'failed', updated_at = NOW() WHERE id = $1", domainID)
		return fmt.Errorf("certbot failed: %w", err)
	}

	log.Printf("ssl: cert issued for %s", hostname)
	s.db.Exec("UPDATE domains SET ssl_status = 'active', updated_at = NOW() WHERE id = $1", domainID)

	return nil
}

// GenerateSSLServerBlock returns an Nginx server block with SSL for a domain.
func GenerateSSLServerBlock(hostname string, port int) string {
	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s", hostname)

	return fmt.Sprintf(`
server {
    listen 80;
    server_name %s;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name %s;

    ssl_certificate %s/fullchain.pem;
    ssl_certificate_key %s/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://host.docker.internal:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_connect_timeout 10s;
        proxy_read_timeout 60s;
    }
}
`, hostname, hostname, certPath, certPath, port)
}

// RenewAll runs certbot renew to refresh any certs close to expiry.
// Called by a cron job (see scripts/certbot-renew.sh).
func RenewAll(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "certbot", "renew", "--quiet")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("certbot renew failed: %s", string(output))
	}
	log.Println("ssl: certbot renew complete")
	return nil
}

// CertExists checks whether a Let's Encrypt cert already exists for a domain.
func CertExists(hostname string) bool {
	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", hostname)
	_, err := os.Stat(certPath)
	return err == nil
}

// HasSSL returns true if the domain has an active cert in the DB.
func HasSSL(db *sql.DB, hostname string) bool {
	var status string
	err := db.QueryRow(
		"SELECT ssl_status FROM domains WHERE hostname = $1", hostname,
	).Scan(&status)
	return err == nil && strings.ToLower(status) == "active"
}
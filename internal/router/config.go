package router

import (
	"fmt"
	"strings"
)

type AppRoute struct {
	Slug          string
	Domain        string
	ContainerPort int
	CustomDomains []string
}

func GenerateNginxConfig(routes []AppRoute) string {
	var b strings.Builder

	b.WriteString("# DeployDock — auto-generated Nginx config\n")
	b.WriteString("# Do not edit manually — regenerated on every deploy\n\n")

	for _, r := range routes {
		b.WriteString(serverBlock(
			fmt.Sprintf("%s.%s", r.Slug, r.Domain),
			r.ContainerPort,
		))
		for _, domain := range r.CustomDomains {
			b.WriteString(serverBlock(domain, r.ContainerPort))
		}
	}

	return b.String()
}

func serverBlock(hostname string, port int) string {
	return fmt.Sprintf(`
server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass http://host.docker.internal:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_connect_timeout 10s;
        proxy_read_timeout 60s;
    }
}
`, hostname, port)
}
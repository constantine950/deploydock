package webhook

import (
	"database/sql"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type LogHandler struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewLogHandler(db *sql.DB, rdb *redis.Client) *LogHandler {
	return &LogHandler{db: db, rdb: rdb}
}

// Stream upgrades the connection to WebSocket and streams logs for a deployment.
// GET /deployments/:id/logs (upgraded to WebSocket)
func (h *LogHandler) Stream(c *websocket.Conn) {
	deploymentID := c.Params("id")
	log.Printf("ws: client connected for deployment %s", deploymentID[:8])

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// 1. Send existing logs from DB first
	rows, err := h.db.QueryContext(ctx, `
		SELECT stream, line, created_at FROM build_logs
		WHERE deployment_id = $1
		ORDER BY id ASC
	`, deploymentID)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("error: failed to load logs"))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var stream, line, createdAt string
		if err := rows.Scan(&stream, &line, &createdAt); err != nil {
			continue
		}
		msg := stream + "|" + line
		if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			return
		}
	}
	rows.Close()

	// 2. Check if deployment is still active — if not, close after history
	var status string
	h.db.QueryRowContext(ctx, "SELECT status FROM deployments WHERE id = $1", deploymentID).Scan(&status)
	if status == "live" || status == "failed" || status == "rolled_back" {
		c.WriteMessage(websocket.TextMessage, []byte("stdout|[stream closed — deployment finished]"))
		return
	}

	// 3. Subscribe to Redis pub/sub for live lines
	channel := "logs:" + deploymentID
	sub := h.rdb.Subscribe(ctx, channel)
	defer sub.Close()

	msgCh := sub.Channel()

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			if err := c.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				log.Printf("ws: client disconnected for %s", deploymentID[:8])
				return
			}
			// Close stream when deployment finishes
			if msg.Payload == "stdout|[deployment finished]" ||
				msg.Payload == "stderr|[deployment failed]" {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// Upgrade middleware — checks if request is a WebSocket upgrade.
func WSUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}
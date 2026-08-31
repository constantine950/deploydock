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

func (h *LogHandler) Stream(c *websocket.Conn) {
	deploymentID := c.Params("id")
	log.Printf("ws: client connected for deployment %s", deploymentID[:8])

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// 1. Send existing logs from DB first
	rows, err := h.db.QueryContext(ctx, `
		SELECT stream, line FROM build_logs
		WHERE deployment_id = $1
		ORDER BY id ASC
	`, deploymentID)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("stdout|error: failed to load logs"))
		return
	}

	for rows.Next() {
		var stream, line string
		if err := rows.Scan(&stream, &line); err != nil {
			continue
		}
		if err := c.WriteMessage(websocket.TextMessage, []byte(stream+"|"+line)); err != nil {
			rows.Close()
			return
		}
	}
	rows.Close()

	// 2. Check deployment status
	var status string
	h.db.QueryRowContext(ctx, "SELECT status FROM deployments WHERE id = $1", deploymentID).Scan(&status)

	if status == "live" || status == "failed" || status == "rolled_back" {
    c.WriteMessage(websocket.TextMessage, []byte("stdout|[deployment finished]"))
    // Keep connection open — wait for client to disconnect
    for {
        _, _, err := c.ReadMessage()
        if err != nil {
            return
        }
    }
}

	// 3. Subscribe to Redis for live lines
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
			if msg.Payload == "stdout|[deployment finished]" ||
				msg.Payload == "stderr|[deployment failed]" {
				time.Sleep(300 * time.Millisecond)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func WSUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}
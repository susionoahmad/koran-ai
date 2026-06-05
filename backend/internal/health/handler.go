package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

// NewHandler instantiates a new Health check HTTP Handler.
func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

// Check handles GET /health, verifying database and Redis connections.
func (h *Handler) Check(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dbStatus := "UP"
	if h.db == nil {
		dbStatus = "DOWN"
	} else if err := h.db.Ping(ctx); err != nil {
		dbStatus = "DOWN"
	}

	redisStatus := "UP"
	if h.rdb == nil {
		redisStatus = "DOWN"
	} else if err := h.rdb.Ping(ctx).Err(); err != nil {
		redisStatus = "DOWN"
	}

	healthy := dbStatus == "UP" && redisStatus == "UP"

	statusCode := http.StatusOK
	message := "OK"
	if !healthy {
		statusCode = http.StatusServiceUnavailable
		message = "Service Unavailable"
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"success": healthy,
		"message": message,
		"data": fiber.Map{
			"service":   "koran-ai",
			"database":  dbStatus,
			"redis":     redisStatus,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

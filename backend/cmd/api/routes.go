package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"koran-ai-backend/internal/health"
	"koran-ai-backend/internal/shared/middleware"
)

// registerRoutes sets up all public and internal API routes.
func registerRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, internalAPIKey string) {
	// Health check (public, no auth)
	healthHandler := health.NewHandler(db, rdb)
	app.Get("/health", healthHandler.Check)

	// Internal API group (protected via X-Internal-Key header)
	internal := app.Group("/internal", middleware.InternalAPIKey(internalAPIKey))
	_ = internal // Future internal routes will be registered here (Phase 3-7)

	// Public API v1 group
	v1 := app.Group("/api/v1")
	_ = v1 // Future public routes will be registered here (Phase 8+)
}

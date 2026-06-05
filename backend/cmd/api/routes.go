package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	aiClient "koran-ai-backend/internal/ai/client"
	aiHandler "koran-ai-backend/internal/ai/handler"
	aiService "koran-ai-backend/internal/ai/service"
	aiWorker "koran-ai-backend/internal/ai/worker"
	articleRepo "koran-ai-backend/internal/article/repository"
	crawlerHandler "koran-ai-backend/internal/crawler/handler"
	crawlerRepo "koran-ai-backend/internal/crawler/repository"
	crawlerRSS "koran-ai-backend/internal/crawler/rss"
	crawlerService "koran-ai-backend/internal/crawler/service"
	"koran-ai-backend/internal/health"
	"koran-ai-backend/internal/shared/config"
	appLogger "koran-ai-backend/internal/shared/logger"
	"koran-ai-backend/internal/shared/middleware"
	"koran-ai-backend/internal/shared/validator"
	"koran-ai-backend/internal/source/handler"
	sourceRepo "koran-ai-backend/internal/source/repository"
	"koran-ai-backend/internal/source/service"
)

// registerRoutes sets up all public and internal API routes.
func registerRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, log appLogger.Logger, cfg *config.Config) {
	// ── Health check (public, no auth) ───────────────────────────────────────
	healthHandler := health.NewHandler(db, rdb)
	app.Get("/health", healthHandler.Check)

	// ── Internal API group (protected via X-Internal-Key header) ─────────────
	internal := app.Group("/internal", middleware.InternalAPIKey(cfg.InternalAPIKey))

	// Source Module
	srcRepo := sourceRepo.NewPostgresRepository(db)

	// Crawler Module
	artRepo := articleRepo.NewPostgresRepository(db)
	logRepo := crawlerRepo.NewCrawlLogRepository(db)
	parser := crawlerRSS.NewParser()
	crawlSvc := crawlerService.NewCrawlerService(srcRepo, artRepo, logRepo, parser)
	crawlHandler := crawlerHandler.NewHandler(crawlSvc)

	internal.Post("/crawler/run/:id", crawlHandler.RunSource)
	internal.Post("/crawler/run-all", crawlHandler.RunAll)
	internal.Get("/crawler/stats", crawlHandler.GetStats)

	// AI Categorization Module
	clientInstance, err := aiClient.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		log.Error("Failed to initialize Gemini Client", zap.Error(err))
		// Log but do not panic so server can boot, though requests to worker will fail.
		// Wait, if it fails to initialize we can still instantiate service/handlers.
	}

	workerInstance := aiWorker.NewAIWorker(artRepo, clientInstance, rdb, log)
	aiSvc := aiService.NewAIService(artRepo, workerInstance)
	aiHdl := aiHandler.NewHandler(aiSvc)

	internal.Post("/ai/process", aiHdl.Process)
	internal.Get("/ai/stats", aiHdl.GetStats)

	// ── Public API v1 group ───────────────────────────────────────────────────
	v1 := app.Group("/api/v1")

	// Source Routes
	srcSvc := service.NewSourceService(srcRepo)
	val := validator.NewValidator()
	srcHandler := handler.NewHandler(srcSvc, val)

	v1.Post("/sources", srcHandler.Create)
	v1.Get("/sources", srcHandler.List)
	v1.Get("/sources/:id", srcHandler.GetByID)
	v1.Put("/sources/:id", srcHandler.Update)
	v1.Delete("/sources/:id", srcHandler.Delete)
}


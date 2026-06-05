package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	"koran-ai-backend/internal/shared/config"
	"koran-ai-backend/internal/shared/database"
	"koran-ai-backend/internal/shared/logger"
	"koran-ai-backend/internal/shared/middleware"
)

func main() {
	// 1. Load configuration
	cfg := config.InitConfig()

	// 2. Initialize structured Zap logger
	log, err := logger.NewLogger(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Sync()
	}()

	log.Info("Starting Koran AI Indonesia API service...", zap.String("env", cfg.AppEnv))

	// 3. Connect to PostgreSQL
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	dbPool, err := database.ConnectPostgres(dbCtx, cfg)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL database", zap.Error(err))
	}
	defer dbPool.Close()
	log.Info("Successfully connected to PostgreSQL connection pool")

	// 4. Connect to Redis
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	redisClient, err := database.ConnectRedis(redisCtx, cfg)
	if err != nil {
		log.Fatal("Failed to connect to Redis server", zap.Error(err))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("Error closing Redis connection", zap.Error(err))
		}
	}()
	log.Info("Successfully connected to Redis client")

	// 5. Initialize Go Fiber application
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	// 6. Register Middlewares (Strict order: RequestID -> Logger -> Recovery)
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger(log))
	app.Use(middleware.Recovery(log))

	// 7. Register all routes
	registerRoutes(app, dbPool, redisClient, cfg.InternalAPIKey)

	// 8. Handle Graceful Shutdown in a separate goroutine
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("Starting HTTP Web Server...", zap.Int("port", cfg.AppPort))
		addr := fmt.Sprintf(":%d", cfg.AppPort)
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP Server listening error", zap.Error(err))
		}
	}()

	// Block until a shutdown signal is received
	sig := <-shutdownChan
	log.Warn("Shutdown signal received, shutting down gracefully", zap.String("signal", sig.String()))

	// Create a timeout context for shutdown sequence
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// Shut down Go Fiber listener
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("HTTP Server shutdown failure", zap.Error(err))
	} else {
		log.Info("HTTP Server listener stopped successfully")
	}

	log.Info("Graceful shutdown sequence complete. Exiting.")
}

package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	"koran-ai-backend/internal/shared/logger"
)

// Logger yields a logging middleware that records HTTP requests.
func Logger(log logger.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		method := c.Method()
		path := c.Path()

		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()
		rid := c.Locals("request_id")

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", duration),
		}

		if rid != nil {
			fields = append(fields, zap.String("request_id", rid.(string)))
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Error("Request failed", fields...)
			return err
		}

		log.Info("Request handled", fields...)
		return nil
	}
}

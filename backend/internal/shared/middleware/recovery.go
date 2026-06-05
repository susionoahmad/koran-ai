package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	"koran-ai-backend/internal/shared/logger"
	"koran-ai-backend/internal/shared/response"
)

// Recovery returns a middleware that handles application panics, logs them, and returns an internal server error response.
func Recovery(log logger.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				stack := debug.Stack()
				rid := c.Locals("request_id")

				fields := []zap.Field{
					zap.Error(err),
					zap.String("stack", string(stack)),
				}
				if rid != nil {
					fields = append(fields, zap.String("request_id", rid.(string)))
				}

				log.Error("System panic recovered", fields...)

				// Return 500 Internal Server Error
				_ = response.Error(c, http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		return c.Next()
	}
}

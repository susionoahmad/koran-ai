package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// RequestID yields a Go Fiber v3 middleware that registers a unique UUID for every request.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		rid := c.Get("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Set("X-Request-ID", rid)
		c.Locals("request_id", rid)
		return c.Next()
	}
}

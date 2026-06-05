package middleware

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"koran-ai-backend/internal/shared/response"
)

// InternalAPIKey returns a middleware that validates the X-Internal-Key header
// for all /internal/* routes.
func InternalAPIKey(apiKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		key := c.Get("X-Internal-Key")
		if key == "" || key != apiKey {
			return response.Error(c, http.StatusUnauthorized, "Unauthorized - Invalid or missing API Key")
		}
		return c.Next()
	}
}

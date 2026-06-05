package response

import "github.com/gofiber/fiber/v3"

type SuccessEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

type PaginatedEnvelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    interface{} `json:"meta"`
}

// JSON sends a standard success response.
func JSON(c fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(SuccessEnvelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error sends a standard error response.
func Error(c fiber.Ctx, status int, message string, errs ...interface{}) error {
	var detail interface{}
	if len(errs) > 0 {
		detail = errs[0]
	}
	return c.Status(status).JSON(ErrorEnvelope{
		Success: false,
		Message: message,
		Errors:  detail,
	})
}

// Paginated sends a standard paginated response.
func Paginated(c fiber.Ctx, status int, data interface{}, meta interface{}) error {
	return c.Status(status).JSON(PaginatedEnvelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

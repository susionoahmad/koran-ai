package validator

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

type Validator interface {
	Validate(i interface{}) map[string]string
}

type structValidator struct {
	v *validator.Validate
}

// NewValidator initializes a Validator implementation.
func NewValidator() Validator {
	return &structValidator{
		v: validator.New(),
	}
}

// Validate validates structs and returns a map of failed fields and their error details.
func (sv *structValidator) Validate(i interface{}) map[string]string {
	err := sv.v.Struct(i)
	if err == nil {
		return nil
	}

	errs := make(map[string]string)
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		errs["error"] = err.Error()
		return errs
	}

	for _, e := range validationErrors {
		// Custom format for error messages
		var tagMsg string
		switch e.Tag() {
		case "required":
			tagMsg = "this field is required"
		case "email":
			tagMsg = "must be a valid email address"
		case "uuid4":
			tagMsg = "must be a valid UUID v4"
		case "url":
			tagMsg = "must be a valid URL"
		case "oneof":
			tagMsg = fmt.Sprintf("must be one of: %s", e.Param())
		case "min":
			tagMsg = fmt.Sprintf("minimum length/value is %s", e.Param())
		case "max":
			tagMsg = fmt.Sprintf("maximum length/value is %s", e.Param())
		default:
			tagMsg = fmt.Sprintf("failed validation: %s", e.Tag())
		}
		errs[e.Field()] = tagMsg
	}

	return errs
}

package validator

import (
	"testing"
)

type TestUser struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18,max=100"`
	URL   string `validate:"url"`
}

func TestValidate_Success(t *testing.T) {
	v := NewValidator()
	user := TestUser{
		Name:  "Jane Doe",
		Email: "jane.doe@example.com",
		Age:   25,
		URL:   "https://example.com",
	}

	errs := v.Validate(user)
	if errs != nil {
		t.Errorf("Expected nil validation errors, got %v", errs)
	}
}

func TestValidate_Failure(t *testing.T) {
	v := NewValidator()
	user := TestUser{
		Name:  "",
		Email: "invalid-email",
		Age:   15,
		URL:   "invalid-url",
	}

	errs := v.Validate(user)
	if errs == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	if msg, exists := errs["Name"]; !exists || msg != "this field is required" {
		t.Errorf("Expected Name error 'this field is required', got '%s'", msg)
	}

	if msg, exists := errs["Email"]; !exists || msg != "must be a valid email address" {
		t.Errorf("Expected Email error 'must be a valid email address', got '%s'", msg)
	}

	if msg, exists := errs["Age"]; !exists || msg != "minimum length/value is 18" {
		t.Errorf("Expected Age error 'minimum length/value is 18', got '%s'", msg)
	}

	if msg, exists := errs["URL"]; !exists || msg != "must be a valid URL" {
		t.Errorf("Expected URL error 'must be a valid URL', got '%s'", msg)
	}
}

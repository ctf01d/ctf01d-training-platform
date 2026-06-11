package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{ErrNotFound, ErrConflict, ErrForbidden, ErrUnauthorized}
	for _, err := range sentinels {
		if err == nil {
			t.Errorf("expected non-nil sentinel error, got nil")
		}
	}
}

func TestSentinelErrorsAreDistinct(t *testing.T) {
	errsList := []error{ErrNotFound, ErrConflict, ErrForbidden, ErrUnauthorized}
	for i := 0; i < len(errsList); i++ {
		for j := i + 1; j < len(errsList); j++ {
			if errors.Is(errsList[i], errsList[j]) {
				t.Errorf("sentinel errors %v and %v should be distinct", errsList[i], errsList[j])
			}
		}
	}
}

func TestSentinelErrorsWrap(t *testing.T) {
	wrapped := fmt.Errorf("user 42: %w", ErrNotFound)
	if !errors.Is(wrapped, ErrNotFound) {
		t.Error("wrapped error should match ErrNotFound via errors.Is")
	}
}

func TestValidationError(t *testing.T) {
	fields := map[string]string{"email": "invalid", "name": "required"}
	ve := NewValidationError(fields)

	if ve.Error() == "" {
		t.Error("ValidationError.Error() should return non-empty string")
	}

	if len(ve.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(ve.Fields))
	}
	if ve.Fields["email"] != "invalid" {
		t.Errorf("expected email=invalid, got %s", ve.Fields["email"])
	}
	if ve.Fields["name"] != "required" {
		t.Errorf("expected name=required, got %s", ve.Fields["name"])
	}
}

func TestValidationErrorAs(t *testing.T) {
	ve := NewValidationError(map[string]string{"foo": "bar"})
	var target *ValidationError
	if !errors.As(ve, &target) {
		t.Error("errors.As should match *ValidationError")
	}
	if target.Fields["foo"] != "bar" {
		t.Error("fields not preserved through errors.As")
	}
}

func TestWrappedValidationErrorAs(t *testing.T) {
	inner := NewValidationError(map[string]string{"x": "y"})
	wrapped := fmt.Errorf("update failed: %w", inner)
	var target *ValidationError
	if !errors.As(wrapped, &target) {
		t.Error("errors.As should unwrap to *ValidationError")
	}
	if target.Fields["x"] != "y" {
		t.Error("fields not preserved through wrapping")
	}
}

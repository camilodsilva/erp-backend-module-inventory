package product

import (
	"errors"
	"testing"
)

func TestProductIdentifier_NewIdentifier_Success(t *testing.T) {
	id, err := NewIdentifier(" 01960e4a-3f2b-7d1c-8e5f-9a0b1c2d3e4f ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "01960e4a-3f2b-7d1c-8e5f-9a0b1c2d3e4f" {
		t.Errorf("expected trimmed id, got %q", id)
	}
}

func TestProductIdentifier_NewIdentifier_Invalid(t *testing.T) {
	_, err := NewIdentifier("invalid")
	if !errors.Is(err, ErrProductIDInvalid) {
		t.Errorf("expected ErrProductIDInvalid, got %v", err)
	}
}

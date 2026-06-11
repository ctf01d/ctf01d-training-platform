package logger

import (
	"testing"
)

func TestNewDevelopment(t *testing.T) {
	l, err := New("development", "debug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	Sync(l)
}

func TestNewProduction(t *testing.T) {
	l, err := New("production", "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	Sync(l)
}

func TestNewInvalidLevel(t *testing.T) {
	_, err := New("development", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}

package auth

import (
	"testing"
	"time"
)

func TestJWT_RoundTrip(t *testing.T) {
	m := NewManager("test-secret", 24)
	token, err := m.Generate(42, "admin", "testuser", "jti1")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.Subject != "42" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "42")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
	if claims.UserName != "testuser" {
		t.Errorf("UserName = %q, want %q", claims.UserName, "testuser")
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	m := NewManager("test-secret", 24)
	m.ttl = -1 * time.Hour

	token, err := m.Generate(1, "guest", "expired", "jti2")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	_, err = m.Parse(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWT_InvalidToken(t *testing.T) {
	m := NewManager("test-secret", 24)

	_, err := m.Parse("invalid.token.string")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestJWT_WrongSecret(t *testing.T) {
	m1 := NewManager("secret1", 24)
	m2 := NewManager("secret2", 24)

	token, err := m1.Generate(1, "guest", "user", "jti4")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	_, err = m2.Parse(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

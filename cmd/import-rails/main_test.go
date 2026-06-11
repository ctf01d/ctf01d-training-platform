package main

import (
	"testing"
	"time"
)

func TestMapRailsUserToParams(t *testing.T) {
	now := time.Now()
	avatar := "https://example.com/avatar.png"
	digest := "$2a$12$somehash"

	ru := RailsUser{
		ID:             1,
		UserName:       "testuser",
		DisplayName:    "Test User",
		Role:           "admin",
		Rating:         42,
		AvatarUrl:      &avatar,
		PasswordDigest: &digest,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	params := MapRailsUserToParams(ru)

	if params.UserName != "testuser" {
		t.Errorf("UserName = %q, want %q", params.UserName, "testuser")
	}
	if params.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", params.DisplayName, "Test User")
	}
	if params.Role != "admin" {
		t.Errorf("Role = %q, want %q", params.Role, "admin")
	}
	if params.Rating != 42 {
		t.Errorf("Rating = %d, want %d", params.Rating, 42)
	}
	if params.AvatarUrl == nil || *params.AvatarUrl != avatar {
		t.Errorf("AvatarUrl = %v, want %q", params.AvatarUrl, avatar)
	}
	if params.PasswordDigest == nil || *params.PasswordDigest != digest {
		t.Errorf("PasswordDigest = %v, want %q", params.PasswordDigest, digest)
	}
}

func TestMapRailsUserToParams_NullableFields(t *testing.T) {
	ru := RailsUser{
		ID:             2,
		UserName:       "minimal",
		DisplayName:    "Minimal User",
		Role:           "guest",
		Rating:         0,
		AvatarUrl:      nil,
		PasswordDigest: nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	params := MapRailsUserToParams(ru)

	if params.AvatarUrl != nil {
		t.Errorf("AvatarUrl = %v, want nil", params.AvatarUrl)
	}
	if params.PasswordDigest != nil {
		t.Errorf("PasswordDigest = %v, want nil", params.PasswordDigest)
	}
}

func TestMapRailsUserToParams_PreservesPasswordDigest(t *testing.T) {
	bcryptHash := "$2a$12$LJ3m4ys3Lg2RqwmMpVr5kuYDFnGMHbOncuWCtRENdQ2JOqM7KrJtG"

	ru := RailsUser{
		UserName:       "rails_user",
		DisplayName:    "Rails User",
		Role:           "player",
		Rating:         100,
		PasswordDigest: &bcryptHash,
	}

	params := MapRailsUserToParams(ru)

	if params.PasswordDigest == nil {
		t.Fatal("PasswordDigest is nil")
	}
	if *params.PasswordDigest != bcryptHash {
		t.Errorf("PasswordDigest = %q, want exact bcrypt hash preserved", *params.PasswordDigest)
	}
}

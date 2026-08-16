package services

import (
	"testing"

	"github.com/google/uuid"
)

func TestJWTServiceRoundTrip(t *testing.T) {
	service := NewJWTService("a-secret-that-is-at-least-32-characters")
	userID := uuid.New()

	token, err := service.Generate(userID, "user@edu.hse.ru", "user")
	if err != nil {
		t.Fatalf("Generate() returned an error: %v", err)
	}
	claims, err := service.Parse(token)
	if err != nil {
		t.Fatalf("Parse() returned an error: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("Subject = %q, want %q", claims.Subject, userID)
	}
	if claims.Email != "user@edu.hse.ru" || claims.Role != "user" {
		t.Errorf("unexpected claims: email=%q role=%q", claims.Email, claims.Role)
	}
}

func TestJWTServiceRejectsDifferentSecret(t *testing.T) {
	token, err := NewJWTService("first-secret-that-is-at-least-32-chars").Generate(uuid.New(), "user@edu.hse.ru", "user")
	if err != nil {
		t.Fatalf("Generate() returned an error: %v", err)
	}

	if _, err := NewJWTService("other-secret-that-is-at-least-32-chars").Parse(token); err == nil {
		t.Fatal("Parse() returned nil error for a token signed with another secret")
	}
}

package services

import (
	"strings"
	"testing"
)

func TestExtractDomainNormalizesEmail(t *testing.T) {
	if got := extractDomain("  User@EDU.HSE.RU  "); got != "edu.hse.ru" {
		t.Errorf("extractDomain() = %q, want edu.hse.ru", got)
	}
}

func TestAllowedDomains(t *testing.T) {
	tests := []struct {
		domain string
		want   bool
	}{
		{domain: "hse.ru", want: true},
		{domain: "edu.hse.ru", want: true},
		{domain: "miem.hse.ru", want: true},
		{domain: "example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			if got := isAllowedDomain(tt.domain); got != tt.want {
				t.Errorf("isAllowedDomain(%q) = %t, want %t", tt.domain, got, tt.want)
			}
		})
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	raw, hash := generateRefreshToken()
	if raw == "" || hash == "" {
		t.Fatal("generateRefreshToken() returned an empty value")
	}
	if raw == hash {
		t.Fatal("refresh token hash must differ from the raw token")
	}
	if len(hash) != 64 || strings.ToLower(hash) != hash {
		t.Errorf("hash = %q, want a lowercase SHA-256 hex string", hash)
	}
}

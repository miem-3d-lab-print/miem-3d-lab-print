package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/repository"
)

const (
	otpExpiresIn      = 10 * time.Minute
	otpMaxAttempts    = 5
	otpBlockDuration  = 15 * time.Minute
	rateLimitWindow   = time.Minute
	rateLimitMaxIP    = 5
	rateLimitMaxEmail = 1
	refreshTokenTTL   = 30 * 24 * time.Hour
)

var allowedDomains = []string{"hse.ru", "edu.hse.ru", "miem.hse.ru"}

type AuthService struct {
	logger           *slog.Logger
	emailService     *EmailService
	otpRepo          repository.OTPRepository
	rateLimitRepo    repository.RateLimitRepository
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtService       *JWTService
}

func NewAuthService(
	logger *slog.Logger,
	emailService *EmailService,
	otpRepo repository.OTPRepository,
	rateLimitRepo repository.RateLimitRepository,
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtService *JWTService,
) *AuthService {
	return &AuthService{
		logger:           logger,
		emailService:     emailService,
		otpRepo:          otpRepo,
		rateLimitRepo:    rateLimitRepo,
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
	}
}

func (s *AuthService) SendOTP(email, ip string) (dto.RequestOTPResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	since := time.Now().Add(-rateLimitWindow)

	// Step 2: rate limit by IP
	ipRequests, err := s.rateLimitRepo.GetRequests("ip", ip, since)
	if err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("check ip rate limit: %w", err)
	}
	if len(ipRequests) >= rateLimitMaxIP {
		retryAfter := retryAfterSeconds(ipRequests[0])
		return dto.RequestOTPResponse{}, &ErrRateLimitIP{RetryAfter: retryAfter}
	}
	if err := s.rateLimitRepo.Record("ip", ip); err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("record ip rate limit: %w", err)
	}

	// Step 3: domain check
	domain := extractDomain(email)
	if !isAllowedDomain(domain) {
		return dto.RequestOTPResponse{}, &ErrInvalidDomain{AllowedDomains: allowedDomains}
	}

	// Step 4: rate limit by email
	emailRequests, err := s.rateLimitRepo.GetRequests("email", email, since)
	if err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("check email rate limit: %w", err)
	}
	if len(emailRequests) >= rateLimitMaxEmail {
		retryAfter := retryAfterSeconds(emailRequests[0])
		return dto.RequestOTPResponse{}, &ErrRateLimitEmail{RetryAfter: retryAfter}
	}

	// Step 5: invalidate existing active OTP
	existing, err := s.otpRepo.FindActive(email)
	if err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("find active otp: %w", err)
	}
	if existing != nil {
		if err := s.otpRepo.Invalidate(existing.ID); err != nil {
			return dto.RequestOTPResponse{}, fmt.Errorf("invalidate otp: %w", err)
		}
	}

	// Step 6: generate 6-digit code
	code, err := generateOTPCode()
	if err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("generate otp: %w", err)
	}

	// Step 7: bcrypt hash
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("hash otp: %w", err)
	}

	// Step 8: persist
	expiresAt := time.Now().Add(otpExpiresIn)
	if err := s.otpRepo.Create(email, string(hash), expiresAt); err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("create otp: %w", err)
	}

	// Step 9: send email — record rate limit only on success so a provider error doesn't burn the user's quota
	s.logger.Info("sending OTP", "email", email)
	if err := s.emailService.SendOTP(email, code); err != nil {
		s.logger.Error("email provider error", "err", err)
		return dto.RequestOTPResponse{}, &ErrEmailProvider{}
	}
	if err := s.rateLimitRepo.Record("email", email); err != nil {
		return dto.RequestOTPResponse{}, fmt.Errorf("record email rate limit: %w", err)
	}

	return dto.RequestOTPResponse{
		Message:   "Код отправлен на почту",
		ExpiresIn: int(otpExpiresIn.Seconds()),
	}, nil
}

func (s *AuthService) VerifyOTP(email, code string) (dto.VerifyOTPResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	// Step 1: find OTP record
	otp, err := s.otpRepo.FindByEmail(email)
	if err != nil {
		return dto.VerifyOTPResponse{}, fmt.Errorf("find otp: %w", err)
	}
	if otp == nil {
		return dto.VerifyOTPResponse{}, &ErrOTPNotFound{}
	}

	// Step 2: check blocked_until
	if otp.BlockedUntil != nil && otp.BlockedUntil.After(time.Now()) {
		return dto.VerifyOTPResponse{}, &ErrOTPLocked{LockedUntil: *otp.BlockedUntil}
	}

	// Step 3: check expiry
	if time.Now().After(otp.ExpiresAt) {
		return dto.VerifyOTPResponse{}, &ErrOTPExpired{}
	}

	// Step 4: compare hash
	if err := bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(code)); err != nil {
		newAttempts := otp.Attempts + 1
		var blockedUntil *time.Time
		if newAttempts >= otpMaxAttempts {
			t := time.Now().Add(otpBlockDuration)
			blockedUntil = &t
		}
		if updateErr := s.otpRepo.UpdateAttempts(otp.ID, newAttempts, blockedUntil); updateErr != nil {
			s.logger.Error("update attempts failed", "err", updateErr)
		}
		return dto.VerifyOTPResponse{}, &ErrInvalidCode{AttemptsLeft: otpMaxAttempts - int(newAttempts)}
	}

	// Step 5: mark OTP used
	if err := s.otpRepo.MarkUsed(otp.ID); err != nil {
		return dto.VerifyOTPResponse{}, fmt.Errorf("mark otp used: %w", err)
	}

	// Step 6: upsert user
	user, err := s.userRepo.Upsert(email)
	if err != nil {
		return dto.VerifyOTPResponse{}, fmt.Errorf("upsert user: %w", err)
	}

	// Step 7: generate tokens
	accessToken, err := s.jwtService.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return dto.VerifyOTPResponse{}, fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, refreshHash := generateRefreshToken()
	expiresAt := time.Now().Add(refreshTokenTTL)
	if _, err := s.refreshTokenRepo.Create(user.ID, refreshHash, expiresAt); err != nil {
		return dto.VerifyOTPResponse{}, fmt.Errorf("create refresh token: %w", err)
	}

	return dto.VerifyOTPResponse{
		AccessToken:     accessToken,
		RefreshToken:    rawRefresh,
		TokenType:       "Bearer",
		ExpiresIn:       int(accessTokenTTL.Seconds()),
		Role:            user.Role,
		ConsentRequired: !user.ConsentGiven,
	}, nil
}

func (s *AuthService) Refresh(rawToken string) (dto.RefreshResponse, error) {
	tokenHash := hashToken(rawToken)

	rt, err := s.refreshTokenRepo.FindByHash(tokenHash)
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("find refresh token: %w", err)
	}
	if rt == nil {
		return dto.RefreshResponse{}, &ErrInvalidRefreshToken{}
	}

	if rt.RevokedAt != nil {
		// reuse-detection: if this was replaced, revoke the whole chain
		if rt.ReplacedByTokenID != nil {
			_ = s.refreshTokenRepo.RevokeAllForUser(rt.UserID)
		}
		return dto.RefreshResponse{}, &ErrRefreshTokenReused{}
	}

	if time.Now().After(rt.ExpiresAt) {
		return dto.RefreshResponse{}, &ErrRefreshTokenExpired{}
	}

	user, err := s.userRepo.FindByID(rt.UserID)
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return dto.RefreshResponse{}, &ErrInvalidRefreshToken{}
	}

	accessToken, err := s.jwtService.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("generate access token: %w", err)
	}

	// rotate refresh token
	rawNew, hashNew := generateRefreshToken()
	newRT, err := s.refreshTokenRepo.Create(user.ID, hashNew, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("create new refresh token: %w", err)
	}
	if err := s.refreshTokenRepo.Revoke(rt.ID, &newRT.ID); err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("revoke old refresh token: %w", err)
	}

	return dto.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rawNew,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

func (s *AuthService) Logout(rawToken string) error {
	tokenHash := hashToken(rawToken)
	rt, err := s.refreshTokenRepo.FindByHash(tokenHash)
	if err != nil {
		return fmt.Errorf("find refresh token: %w", err)
	}
	if rt == nil || rt.RevokedAt != nil {
		return &ErrInvalidRefreshToken{}
	}
	return s.refreshTokenRepo.Revoke(rt.ID, nil)
}

func extractDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func isAllowedDomain(domain string) bool {
	for _, d := range allowedDomains {
		if d == domain {
			return true
		}
	}
	return false
}

func retryAfterSeconds(oldest time.Time) int {
	v := int(rateLimitWindow.Seconds()) - int(time.Since(oldest).Seconds())
	if v < 1 {
		return 1
	}
	return v
}

func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateRefreshToken() (raw, hash string) {
	raw = uuid.New().String()
	hash = hashToken(raw)
	return
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

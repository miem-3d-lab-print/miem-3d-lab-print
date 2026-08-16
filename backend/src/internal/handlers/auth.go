package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/apierr"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
	logger      *slog.Logger
	trustProxy  bool
}

func NewAuthHandler(authService *services.AuthService, logger *slog.Logger, trustProxy bool) *AuthHandler {
	return &AuthHandler{authService: authService, logger: logger, trustProxy: trustProxy}
}

func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/request-otp", h.RequestOTP)
	mux.HandleFunc("POST /api/auth/verify-otp", h.VerifyOTP)
	mux.HandleFunc("POST /api/auth/refresh", h.Refresh)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
}

// RequestOTP отправляет OTP-код на указанный email.
//
//	@Summary		Запросить OTP-код
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RequestOTPRequest	true	"Email адрес"
//	@Success		200		{object}	dto.RequestOTPResponse
//	@Failure		400		{object}	apierr.ErrorResponse
//	@Failure		429		{object}	apierr.ErrorResponse	"Превышен лимит запросов"
//	@Failure		500		{object}	apierr.ErrorResponse
//	@Router			/api/auth/request-otp [post]
func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req dto.RequestOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "INVALID_EMAIL_FORMAT", "Некорректный формат email", nil)
		return
	}
	if err := validate.Struct(req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "INVALID_EMAIL_FORMAT", "Некорректный формат email", nil)
		return
	}

	ip := extractIP(r, h.trustProxy)
	resp, err := h.authService.SendOTP(req.Email, ip)
	if err != nil {
		var errIP *services.ErrRateLimitIP
		var errEmail *services.ErrRateLimitEmail
		var errDomain *services.ErrInvalidDomain
		var errProvider *services.ErrEmailProvider

		switch {
		case errors.As(err, &errIP):
			apierr.Write(w, http.StatusTooManyRequests, "RATE_LIMIT_IP",
				"Превышен лимит запросов с вашего IP",
				map[string]int{"retry_after": errIP.RetryAfter})
		case errors.As(err, &errEmail):
			apierr.Write(w, http.StatusTooManyRequests, "RATE_LIMIT_EMAIL",
				"Повторный запрос кода слишком рано",
				map[string]int{"retry_after": errEmail.RetryAfter})
		case errors.As(err, &errDomain):
			apierr.Write(w, http.StatusBadRequest, "INVALID_DOMAIN",
				"Домен email не входит в список допустимых",
				map[string][]string{"allowed_domains": errDomain.AllowedDomains})
		case errors.As(err, &errProvider):
			h.logger.Error("email provider error", "email", req.Email)
			apierr.Write(w, http.StatusInternalServerError, "EMAIL_PROVIDER_ERROR",
				"Не удалось отправить письмо", nil)
		default:
			h.logger.Error("unexpected error in SendOTP", "email", req.Email, "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка сервера", nil)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// VerifyOTP проверяет OTP-код и выдаёт токены.
//
//	@Summary	Подтвердить OTP-код
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		dto.VerifyOTPRequest	true	"Email + OTP-код"
//	@Success	200		{object}	dto.VerifyOTPResponse
//	@Failure	400		{object}	apierr.ErrorResponse	"Неверный или истёкший код"
//	@Failure	404		{object}	apierr.ErrorResponse	"OTP не найден"
//	@Failure	423		{object}	apierr.ErrorResponse	"Слишком много попыток"
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "INVALID_EMAIL_FORMAT", "Некорректный формат запроса", nil)
		return
	}
	if err := validate.Struct(req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "INVALID_EMAIL_FORMAT", "Некорректный формат запроса", nil)
		return
	}

	resp, err := h.authService.VerifyOTP(req.Email, req.Code)
	if err != nil {
		var errNotFound *services.ErrOTPNotFound
		var errLocked *services.ErrOTPLocked
		var errExpired *services.ErrOTPExpired
		var errInvalid *services.ErrInvalidCode

		switch {
		case errors.As(err, &errNotFound):
			apierr.Write(w, http.StatusNotFound, "OTP_NOT_FOUND",
				"Код не найден или уже использован", nil)
		case errors.As(err, &errLocked):
			apierr.Write(w, http.StatusLocked, "LOCKED",
				"Превышено количество попыток, повторите позже",
				map[string]string{"locked_until": errLocked.LockedUntil.UTC().Format("2006-01-02T15:04:05Z")})
		case errors.As(err, &errExpired):
			apierr.Write(w, http.StatusBadRequest, "CODE_EXPIRED",
				"Срок действия кода истёк", nil)
		case errors.As(err, &errInvalid):
			apierr.Write(w, http.StatusBadRequest, "INVALID_CODE",
				"Неверный код",
				map[string]int{"attempts_left": errInvalid.AttemptsLeft})
		default:
			h.logger.Error("verify otp error", "email", req.Email, "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка сервера", nil)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Refresh обновляет пару access/refresh токенов.
//
//	@Summary	Обновить токены
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		dto.RefreshRequest	true	"Refresh token"
//	@Success	200		{object}	dto.RefreshResponse
//	@Failure	401		{object}	apierr.ErrorResponse	"Невалидный/просроченный/отозванный token"
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		apierr.Write(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN",
			"Refresh token не передан", nil)
		return
	}

	resp, err := h.authService.Refresh(req.RefreshToken)
	if err != nil {
		var errInvalid *services.ErrInvalidRefreshToken
		var errExpired *services.ErrRefreshTokenExpired
		var errRevoked *services.ErrRefreshTokenRevoked
		var errReused *services.ErrRefreshTokenReused

		switch {
		case errors.As(err, &errInvalid):
			apierr.Write(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN",
				"Refresh token недействителен", nil)
		case errors.As(err, &errExpired):
			apierr.Write(w, http.StatusUnauthorized, "REFRESH_TOKEN_EXPIRED",
				"Refresh token истёк", nil)
		case errors.As(err, &errRevoked):
			apierr.Write(w, http.StatusUnauthorized, "REFRESH_TOKEN_REVOKED",
				"Refresh token отозван", nil)
		case errors.As(err, &errReused):
			apierr.Write(w, http.StatusUnauthorized, "REFRESH_TOKEN_REUSED",
				"Обнаружено повторное использование отозванного токена", nil)
		default:
			h.logger.Error("refresh error", "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка сервера", nil)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Logout отзывает refresh token.
//
//	@Summary	Выйти (отозвать refresh token)
//	@Tags		auth
//	@Accept		json
//	@Param		request	body	dto.LogoutRequest	true	"Refresh token"
//	@Success	204
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		apierr.Write(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN",
			"Refresh token не передан", nil)
		return
	}

	if err := h.authService.Logout(req.RefreshToken); err != nil {
		var errInvalid *services.ErrInvalidRefreshToken
		if errors.As(err, &errInvalid) {
			apierr.Write(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN",
				"Refresh token недействителен или уже отозван", nil)
			return
		}
		h.logger.Error("logout error", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"Внутренняя ошибка сервера", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractIP returns the client IP.
// When trustProxy is true the rightmost entry in X-Forwarded-For is used
// (it was added by the trusted reverse-proxy and cannot be spoofed by the client).
// When trustProxy is false only RemoteAddr is trusted.
func extractIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

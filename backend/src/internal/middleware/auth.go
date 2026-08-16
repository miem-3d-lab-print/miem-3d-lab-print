package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/apierr"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/repository"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

type contextKey int

const (
	contextKeyUserID contextKey = iota
	contextKeyUserEmail
	contextKeyUserRole
)

func Auth(jwtService *services.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				apierr.Write(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация", nil)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtService.Parse(tokenStr)
			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					apierr.Write(w, http.StatusUnauthorized, "TOKEN_EXPIRED",
						"Токен истёк, обновите через /auth/refresh", nil)
					return
				}
				apierr.Write(w, http.StatusUnauthorized, "UNAUTHORIZED", "Недействительный токен", nil)
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				apierr.Write(w, http.StatusUnauthorized, "UNAUTHORIZED", "Недействительный токен", nil)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, contextKeyUserID, userID)
			ctx = context.WithValue(ctx, contextKeyUserEmail, claims.Email)
			ctx = context.WithValue(ctx, contextKeyUserRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireConsent(userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromCtx(r.Context())
			user, err := userRepo.FindByID(userID)
			if err != nil || user == nil {
				apierr.Write(w, http.StatusUnauthorized, "UNAUTHORIZED", "Пользователь не найден", nil)
				return
			}
			if !user.ConsentGiven {
				apierr.Write(w, http.StatusForbidden, "CONSENT_REQUIRED",
					"Необходимо подтвердить согласие на обработку персональных данных", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserRoleFromCtx(r.Context()) != models.UserRoleAdmin {
			apierr.Write(w, http.StatusForbidden, "FORBIDDEN", "Доступ запрещён", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserIDFromCtx(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(contextKeyUserID).(uuid.UUID); ok {
		return v
	}
	return uuid.UUID{}
}

func UserRoleFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyUserRole).(string); ok {
		return v
	}
	return ""
}

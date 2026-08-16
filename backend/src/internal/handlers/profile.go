package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/apierr"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/middleware"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

// swagTypes anchors dto types referenced in swagger annotations.
var _ = dto.PatchProfileRequest{}

type ProfileHandler struct {
	profileService *services.ProfileService
	logger         *slog.Logger
}

func NewProfileHandler(profileService *services.ProfileService, logger *slog.Logger) *ProfileHandler {
	return &ProfileHandler{profileService: profileService, logger: logger}
}

func (h *ProfileHandler) Register(mux *http.ServeMux, authMW func(http.Handler) http.Handler, consentMW func(http.Handler) http.Handler) {
	mux.Handle("GET /api/profile", authMW(http.HandlerFunc(h.GetProfile)))
	mux.Handle("PATCH /api/profile", authMW(consentMW(http.HandlerFunc(h.PatchProfile))))
	mux.Handle("POST /api/profile/consent", authMW(http.HandlerFunc(h.GiveConsent)))
}

// GetProfile возвращает профиль текущего пользователя.
//
//	@Summary	Получить профиль
//	@Tags		profile
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	dto.ProfileResponse
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	404	{object}	apierr.ErrorResponse	"Профиль не найден"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/profile [get]
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	resp, err := h.profileService.GetProfile(userID)
	if err != nil {
		var e *services.ErrProfileNotFound
		if errors.As(err, &e) {
			apierr.Write(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "Профиль не найден", nil)
			return
		}
		h.logger.Error("get profile", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// PatchProfile обновляет поля профиля (full_name, telegram, max).
//
//	@Summary	Обновить профиль
//	@Tags		profile
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		request	body		dto.PatchProfileRequest	true	"Изменяемые поля"
//	@Success	200		{object}	dto.ProfileResponse
//	@Failure	400		{object}	apierr.ErrorResponse	"Ошибка валидации"
//	@Failure	401		{object}	apierr.ErrorResponse
//	@Failure	403		{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	404		{object}	apierr.ErrorResponse
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/profile [patch]
func (h *ProfileHandler) PatchProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		apierr.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "Некорректное тело запроса", nil)
		return
	}
	if !json.Valid(body) {
		apierr.Write(w, http.StatusBadRequest, "INVALID_REQUEST", "Некорректное тело запроса", nil)
		return
	}

	resp, err := h.profileService.PatchProfile(userID, body)
	if err != nil {
		var eField *services.ErrFieldNotEditable
		var eContact *services.ErrContactRequired
		var eName *services.ErrInvalidFullName
		var eNotFound *services.ErrProfileNotFound

		switch {
		case errors.As(err, &eField):
			apierr.Write(w, http.StatusBadRequest, "FIELD_NOT_EDITABLE",
				"Поле недоступно для изменения",
				map[string]string{"field": eField.Field})
		case errors.As(err, &eContact):
			apierr.Write(w, http.StatusBadRequest, "CONTACT_REQUIRED",
				"Необходимо указать хотя бы один контакт (Telegram или MAX)", nil)
		case errors.As(err, &eName):
			apierr.Write(w, http.StatusBadRequest, "INVALID_FULL_NAME",
				"ФИО не может быть пустым", nil)
		case errors.As(err, &eNotFound):
			apierr.Write(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "Профиль не найден", nil)
		default:
			h.logger.Error("patch profile", "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GiveConsent даёт согласие на обработку персональных данных.
//
//	@Summary	Дать согласие на обработку данных
//	@Tags		profile
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	dto.ConsentResponse
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/profile/consent [post]
func (h *ProfileHandler) GiveConsent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	resp, err := h.profileService.GiveConsent(userID)
	if err != nil {
		h.logger.Error("give consent", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

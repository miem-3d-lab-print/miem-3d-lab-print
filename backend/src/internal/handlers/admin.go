package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/apierr"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

type AdminHandler struct {
	adminService *services.AdminService
	statsService *services.StatsService
	logger       *slog.Logger
}

func NewAdminHandler(adminService *services.AdminService, statsService *services.StatsService, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{adminService: adminService, statsService: statsService, logger: logger}
}

func (h *AdminHandler) Register(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	mux.Handle("GET /api/admin/users", authMW(adminMW(http.HandlerFunc(h.SearchUsers))))
	mux.Handle("PATCH /api/admin/users/{id}/role", authMW(adminMW(http.HandlerFunc(h.SetRole))))
	mux.Handle("GET /api/admin/stats", authMW(adminMW(http.HandlerFunc(h.Stats))))
}

// SearchUsers выполняет поиск пользователей по email.
//
//	@Summary	Поиск пользователей по email (admin)
//	@Tags		admin-users
//	@Produce	json
//	@Security	BearerAuth
//	@Param		email	query		string	true	"Минимум 3 символа"	minLength(3)
//	@Success	200		{object}	dto.AdminUsersResponse
//	@Failure	400		{object}	apierr.ErrorResponse	"Запрос слишком короткий"
//	@Failure	401		{object}	apierr.ErrorResponse
//	@Failure	403		{object}	apierr.ErrorResponse
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/admin/users [get]
func (h *AdminHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	emailQuery := r.URL.Query().Get("email")
	resp, err := h.adminService.SearchUsers(emailQuery)
	if err != nil {
		var e *services.ErrQueryTooShort
		if errors.As(err, &e) {
			apierr.Write(w, http.StatusBadRequest, "QUERY_TOO_SHORT",
				"Запрос слишком короткий (минимум 3 символа)",
				map[string]int{"min_length": 3})
			return
		}
		h.logger.Error("search users", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// SetRole изменяет роль пользователя.
//
//	@Summary	Изменить роль пользователя (admin)
//	@Tags		admin-users
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string				true	"ID пользователя"	format(uuid)
//	@Param		request	body		dto.SetRoleRequest	true	"Новая роль"
//	@Success	200		{object}	dto.SetRoleResponse
//	@Failure	400		{object}	apierr.ErrorResponse	"Недопустимое значение роли"
//	@Failure	401		{object}	apierr.ErrorResponse
//	@Failure	403		{object}	apierr.ErrorResponse
//	@Failure	404		{object}	apierr.ErrorResponse
//	@Failure	409		{object}	apierr.ErrorResponse	"Нельзя снять роль с последнего администратора"
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/admin/users/{id}/role [patch]
func (h *AdminHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "USER_NOT_FOUND", "Пользователь не найден", nil)
		return
	}
	var req dto.SetRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное тело запроса", nil)
		return
	}

	resp, err := h.adminService.SetRole(id, req.Role)
	if err != nil {
		var eNF *services.ErrUserNotFound
		var eRole *services.ErrInvalidRole
		var eLast *services.ErrLastAdmin
		switch {
		case errors.As(err, &eNF):
			apierr.Write(w, http.StatusNotFound, "USER_NOT_FOUND", "Пользователь не найден", nil)
		case errors.As(err, &eRole):
			apierr.Write(w, http.StatusBadRequest, "INVALID_ROLE", "Недопустимое значение роли", nil)
		case errors.As(err, &eLast):
			apierr.Write(w, http.StatusConflict, "LAST_ADMIN",
				"Нельзя снять роль с последнего администратора", nil)
		default:
			h.logger.Error("set role", "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Stats возвращает статистику за период.
//
//	@Summary	Статистика (admin)
//	@Tags		admin-stats
//	@Produce	json
//	@Security	BearerAuth
//	@Param		date_from	query		string	false	"Начало периода (YYYY-MM-DD)"
//	@Param		date_to		query		string	false	"Конец периода (YYYY-MM-DD)"
//	@Success	200			{object}	dto.StatsResponse
//	@Failure	400			{object}	apierr.ErrorResponse	"Некорректный период"
//	@Failure	401			{object}	apierr.ErrorResponse
//	@Failure	403			{object}	apierr.ErrorResponse
//	@Failure	500			{object}	apierr.ErrorResponse
//	@Router		/api/admin/stats [get]
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var dateFrom, dateTo *time.Time

	if v := q.Get("date_from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			apierr.Write(w, http.StatusBadRequest, "INVALID_PERIOD", "Некорректный формат date_from", nil)
			return
		}
		dateFrom = &t
	}
	if v := q.Get("date_to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			apierr.Write(w, http.StatusBadRequest, "INVALID_PERIOD", "Некорректный формат date_to", nil)
			return
		}
		end := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		dateTo = &end
	}

	resp, err := h.statsService.GetStats(dateFrom, dateTo)
	if err != nil {
		var e *services.ErrInvalidPeriod
		if errors.As(err, &e) {
			apierr.Write(w, http.StatusBadRequest, "INVALID_PERIOD", "Некорректный период", nil)
			return
		}
		h.logger.Error("get stats", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

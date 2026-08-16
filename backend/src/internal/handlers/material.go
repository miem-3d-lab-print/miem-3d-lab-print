package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/apierr"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

type MaterialHandler struct {
	materialService *services.MaterialService
	logger          *slog.Logger
}

func NewMaterialHandler(materialService *services.MaterialService, logger *slog.Logger) *MaterialHandler {
	return &MaterialHandler{materialService: materialService, logger: logger}
}

func (h *MaterialHandler) Register(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	consentMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	mux.Handle("GET /api/materials", authMW(consentMW(http.HandlerFunc(h.ListActive))))
	mux.Handle("GET /api/admin/materials", authMW(adminMW(http.HandlerFunc(h.ListAll))))
	mux.Handle("POST /api/admin/materials", authMW(adminMW(http.HandlerFunc(h.Create))))
	mux.Handle("PATCH /api/admin/materials/{id}", authMW(adminMW(http.HandlerFunc(h.Update))))
	mux.Handle("POST /api/admin/materials/{id}/colors", authMW(adminMW(http.HandlerFunc(h.CreateColor))))
	mux.Handle("PATCH /api/admin/materials/{id}/colors/{color_id}", authMW(adminMW(http.HandlerFunc(h.UpdateColor))))
}

// ListActive возвращает список активных материалов с цветами.
//
//	@Summary	Список доступных материалов
//	@Tags		materials
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string][]dto.MaterialResponse	"items"
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/materials [get]
func (h *MaterialHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	items, err := h.materialService.ListActive()
	if err != nil {
		h.logger.Error("list active materials", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// ListAll возвращает все материалы включая неактивные (admin).
//
//	@Summary	Все материалы (admin)
//	@Tags		admin-materials
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string][]dto.MaterialResponse	"items"
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/admin/materials [get]
func (h *MaterialHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.materialService.ListAll()
	if err != nil {
		h.logger.Error("list all materials", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// Create создаёт новый материал.
//
//	@Summary	Создать материал (admin)
//	@Tags		admin-materials
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		request	body		dto.CreateMaterialRequest	true	"Данные материала"
//	@Success	201		{object}	dto.MaterialResponse
//	@Failure	401		{object}	apierr.ErrorResponse
//	@Failure	403		{object}	apierr.ErrorResponse
//	@Failure	409		{object}	apierr.ErrorResponse	"Материал с таким названием уже существует"
//	@Failure	422		{object}	apierr.ErrorResponse
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/admin/materials [post]
func (h *MaterialHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		apierr.Write(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректное тело запроса", nil)
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	mat, err := h.materialService.Create(req.Name, req.Description, isActive)
	if err != nil {
		var e *services.ErrMaterialNameExists
		if errors.As(err, &e) {
			apierr.Write(w, http.StatusConflict, "MATERIAL_NAME_EXISTS",
				"Материал с таким названием уже существует",
				map[string]string{"existing_id": e.ExistingID})
			return
		}
		h.logger.Error("create material", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusCreated, mat)
}

// Update обновляет материал.
//
//	@Summary	Обновить материал (admin)
//	@Tags		admin-materials
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string					true	"ID материала"	format(uuid)
//	@Param		request	body		dto.PatchMaterialRequest	true	"Изменяемые поля"
//	@Success	200		{object}	dto.MaterialResponse
//	@Failure	400		{object}	apierr.ErrorResponse
//	@Failure	401		{object}	apierr.ErrorResponse
//	@Failure	403		{object}	apierr.ErrorResponse
//	@Failure	404		{object}	apierr.ErrorResponse
//	@Failure	409		{object}	apierr.ErrorResponse	"Название уже занято"
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/admin/materials/{id} [patch]
func (h *MaterialHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "MATERIAL_NOT_FOUND", "Материал не найден", nil)
		return
	}
	var req dto.PatchMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное тело запроса", nil)
		return
	}
	mat, err := h.materialService.Update(id, req.Name, req.Description, req.IsActive)
	if err != nil {
		var eNF *services.ErrMaterialNotFound
		var eExists *services.ErrMaterialNameExists
		switch {
		case errors.As(err, &eNF):
			apierr.Write(w, http.StatusNotFound, "MATERIAL_NOT_FOUND", "Материал не найден", nil)
		case errors.As(err, &eExists):
			apierr.Write(w, http.StatusConflict, "MATERIAL_NAME_EXISTS",
				"Материал с таким названием уже существует", nil)
		default:
			h.logger.Error("update material", "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, mat)
}

// CreateColor добавляет цвет к материалу.
//
//	@Summary	Добавить цвет к материалу (admin)
//	@Tags		admin-materials
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string					true	"ID материала"	format(uuid)
//	@Param		request	body		dto.CreateColorRequest	true	"Данные цвета"
//	@Success	201		{object}	dto.ColorResponse
//	@Failure	401		{object}	apierr.ErrorResponse
//	@Failure	403		{object}	apierr.ErrorResponse
//	@Failure	404		{object}	apierr.ErrorResponse
//	@Failure	409		{object}	apierr.ErrorResponse	"Цвет с таким названием уже существует"
//	@Failure	422		{object}	apierr.ErrorResponse
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/admin/materials/{id}/colors [post]
func (h *MaterialHandler) CreateColor(w http.ResponseWriter, r *http.Request) {
	materialID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "MATERIAL_NOT_FOUND", "Материал не найден", nil)
		return
	}
	var req dto.CreateColorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		apierr.Write(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректное тело запроса", nil)
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	color, err := h.materialService.CreateColor(materialID, req.Name, isActive)
	if err != nil {
		var eNF *services.ErrMaterialNotFound
		var eExists *services.ErrColorNameExists
		switch {
		case errors.As(err, &eNF):
			apierr.Write(w, http.StatusNotFound, "MATERIAL_NOT_FOUND", "Материал не найден", nil)
		case errors.As(err, &eExists):
			apierr.Write(w, http.StatusConflict, "COLOR_NAME_EXISTS", "Цвет с таким названием уже существует", nil)
		default:
			h.logger.Error("create color", "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		}
		return
	}
	writeJSON(w, http.StatusCreated, color)
}

// UpdateColor обновляет цвет материала.
//
//	@Summary	Обновить цвет (admin)
//	@Tags		admin-materials
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id			path		string				true	"ID материала"	format(uuid)
//	@Param		color_id	path		string				true	"ID цвета"		format(uuid)
//	@Param		request		body		dto.PatchColorRequest	true	"Изменяемые поля"
//	@Success	200			{object}	dto.ColorResponse
//	@Failure	400			{object}	apierr.ErrorResponse
//	@Failure	401			{object}	apierr.ErrorResponse
//	@Failure	403			{object}	apierr.ErrorResponse
//	@Failure	404			{object}	apierr.ErrorResponse
//	@Failure	409			{object}	apierr.ErrorResponse	"Название уже занято"
//	@Failure	500			{object}	apierr.ErrorResponse
//	@Router		/api/admin/materials/{id}/colors/{color_id} [patch]
func (h *MaterialHandler) UpdateColor(w http.ResponseWriter, r *http.Request) {
	materialID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "MATERIAL_NOT_FOUND", "Материал не найден", nil)
		return
	}
	colorID, err := uuid.Parse(r.PathValue("color_id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "COLOR_NOT_FOUND", "Цвет не найден", nil)
		return
	}
	var req dto.PatchColorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное тело запроса", nil)
		return
	}
	color, err := h.materialService.UpdateColor(materialID, colorID, req.Name, req.IsActive)
	if err != nil {
		var eMatNF *services.ErrMaterialNotFound
		var eColNF *services.ErrColorNotFound
		var eExists *services.ErrColorNameExists
		switch {
		case errors.As(err, &eMatNF):
			apierr.Write(w, http.StatusNotFound, "MATERIAL_NOT_FOUND", "Материал не найден", nil)
		case errors.As(err, &eColNF):
			apierr.Write(w, http.StatusNotFound, "COLOR_NOT_FOUND", "Цвет не найден", nil)
		case errors.As(err, &eExists):
			apierr.Write(w, http.StatusConflict, "COLOR_NAME_EXISTS", "Цвет с таким названием уже существует", nil)
		default:
			h.logger.Error("update color", "err", err)
			apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		}
		return
	}
	writeJSON(w, http.StatusOK, color)
}

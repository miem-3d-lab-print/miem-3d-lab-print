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
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/middleware"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

type ApplicationHandler struct {
	appService *services.ApplicationService
	logger     *slog.Logger
}

func NewApplicationHandler(appService *services.ApplicationService, logger *slog.Logger) *ApplicationHandler {
	return &ApplicationHandler{appService: appService, logger: logger}
}

func (h *ApplicationHandler) Register(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	consentMW func(http.Handler) http.Handler,
	adminMW func(http.Handler) http.Handler,
) {
	ac := func(handler http.Handler) http.Handler { return authMW(consentMW(handler)) }

	mux.Handle("GET /api/applications", ac(http.HandlerFunc(h.List)))
	mux.Handle("POST /api/applications", ac(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/applications/{id}", ac(http.HandlerFunc(h.Get)))
	mux.Handle("PATCH /api/applications/{id}/cancel", ac(http.HandlerFunc(h.Cancel)))
	mux.Handle("POST /api/applications/{id}/files", ac(http.HandlerFunc(h.UploadFile)))
	mux.Handle("GET /api/applications/{id}/files/{file_id}", ac(http.HandlerFunc(h.DownloadFile)))

	mux.Handle("GET /api/admin/applications", authMW(adminMW(http.HandlerFunc(h.AdminList))))
	mux.Handle("GET /api/admin/applications/{id}", authMW(adminMW(http.HandlerFunc(h.AdminGet))))
	mux.Handle("PATCH /api/admin/applications/{id}/status", authMW(adminMW(http.HandlerFunc(h.AdminChangeStatus))))
	mux.Handle("GET /api/admin/applications/{id}/files/{file_id}", authMW(adminMW(http.HandlerFunc(h.AdminDownloadFile))))
}

// List возвращает список заявок текущего пользователя с пагинацией.
//
//	@Summary	Список заявок пользователя
//	@Tags		applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		status		query		string	false	"Фильтр по статусу"	Enums(new,in_review,printing,ready,issued,rejected,cancelled)
//	@Param		page		query		int		false	"Страница"			default(1)
//	@Param		per_page	query		int		false	"Записей на странице"	default(20)	maximum(100)
//	@Success	200	{object}	dto.ApplicationListResponse
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/applications [get]
func (h *ApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	status := r.URL.Query().Get("status")
	page, perPage := parsePagination(r)

	resp, err := h.appService.ListByUser(userID, status, page, perPage)
	if err != nil {
		h.logger.Error("list applications", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Create создаёт новую заявку на 3D-печать.
//
//	@Summary	Создать заявку
//	@Tags		applications
//	@Accept		mpfd
//	@Produce	json
//	@Security	BearerAuth
//	@Param		position		formData	string	true	"Должность / группа заявителя"
//	@Param		purpose			formData	string	true	"Цель печати"
//	@Param		material_id		formData	string	true	"UUID материала"	format(uuid)
//	@Param		color_matters	formData	boolean	true	"Важен ли цвет"
//	@Param		color_id		formData	string	false	"UUID цвета (обязателен если color_matters=true)"	format(uuid)
//	@Param		desired_date	formData	string	true	"Желаемая дата получения (YYYY-MM-DD)"
//	@Param		comment			formData	string	false	"Комментарий"
//	@Param		files[]			formData	file	false	"Файлы моделей (STL / STEP / 3MF, до 20 МБ каждый)"
//	@Success	201	{object}	dto.CreateApplicationResponse
//	@Failure	400	{object}	apierr.ErrorResponse
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	409	{object}	apierr.ErrorResponse	"Профиль не заполнен / лимит заявок / материал недоступен"
//	@Failure	413	{object}	apierr.ErrorResponse	"Файл слишком большой"
//	@Failure	422	{object}	apierr.ErrorResponse	"Ошибка валидации"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/applications [post]
func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		apierr.Write(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный multipart запрос", nil)
		return
	}

	userID := middleware.UserIDFromCtx(r.Context())

	position := r.FormValue("position")
	purpose := r.FormValue("purpose")
	materialIDStr := r.FormValue("material_id")
	colorMattersStr := r.FormValue("color_matters")
	colorIDStr := r.FormValue("color_id")
	desiredDateStr := r.FormValue("desired_date")
	commentStr := r.FormValue("comment")

	if position == "" || purpose == "" || materialIDStr == "" || colorMattersStr == "" || desiredDateStr == "" {
		apierr.Write(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Заполните все обязательные поля", nil)
		return
	}

	materialID, err := uuid.Parse(materialIDStr)
	if err != nil {
		apierr.Write(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректный material_id", nil)
		return
	}

	colorMatters := colorMattersStr == "true"
	var colorID *uuid.UUID
	if colorIDStr != "" {
		cid, err := uuid.Parse(colorIDStr)
		if err != nil {
			apierr.Write(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректный color_id", nil)
			return
		}
		colorID = &cid
	}

	desiredDate, err := time.Parse("2006-01-02", desiredDateStr)
	if err != nil {
		apierr.Write(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Некорректный формат desired_date", nil)
		return
	}

	var comment *string
	if commentStr != "" {
		comment = &commentStr
	}

	files := r.MultipartForm.File["files[]"]

	resp, err := h.appService.Create(userID, position, purpose, materialID, colorMatters, colorID, desiredDate, comment, files)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// Get возвращает детали заявки текущего пользователя.
//
//	@Summary	Получить заявку
//	@Tags		applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"ID заявки"	format(uuid)
//	@Success	200	{object}	dto.ApplicationDetailUser
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	404	{object}	apierr.ErrorResponse
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/applications/{id} [get]
func (h *ApplicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
		return
	}
	resp, err := h.appService.GetByIDAndUser(id, userID)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Cancel отменяет заявку (только пока статус new).
//
//	@Summary	Отменить заявку
//	@Tags		applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"ID заявки"	format(uuid)
//	@Success	200	{object}	dto.CancelApplicationResponse
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	404	{object}	apierr.ErrorResponse
//	@Failure	409	{object}	apierr.ErrorResponse	"Отмена невозможна"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/applications/{id}/cancel [patch]
func (h *ApplicationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
		return
	}
	resp, err := h.appService.CancelByUser(id, userID)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// UploadFile добавляет файл к заявке (только пока статус new).
//
//	@Summary	Добавить файл к заявке
//	@Tags		applications
//	@Accept		mpfd
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string	true	"ID заявки"	format(uuid)
//	@Param		file	formData	file	true	"Файл модели (STL / STEP / 3MF, до 20 МБ)"
//	@Success	201	{object}	dto.UploadFileResponse
//	@Failure	400	{object}	apierr.ErrorResponse	"Файл не передан или неверный формат"
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	404	{object}	apierr.ErrorResponse
//	@Failure	409	{object}	apierr.ErrorResponse	"Заявка не в статусе new"
//	@Failure	413	{object}	apierr.ErrorResponse	"Файл слишком большой"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/applications/{id}/files [post]
func (h *ApplicationHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	appID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
		return
	}

	if err := r.ParseMultipartForm(25 << 20); err != nil {
		apierr.Write(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный multipart запрос", nil)
		return
	}

	_, fh, err := r.FormFile("file")
	if err != nil {
		apierr.Write(w, http.StatusBadRequest, "FILES_REQUIRED", "Файл не передан", nil)
		return
	}

	resp, err := h.appService.UploadFileToApp(appID, userID, fh)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// DownloadFile перенаправляет на presigned URL файла.
//
//	@Summary	Скачать файл (редирект на presigned URL)
//	@Tags		applications
//	@Security	BearerAuth
//	@Param		id		path	string	true	"ID заявки"	format(uuid)
//	@Param		file_id	path	string	true	"ID файла"	format(uuid)
//	@Success	302
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse	"Требуется согласие"
//	@Failure	404	{object}	apierr.ErrorResponse
//	@Failure	410	{object}	apierr.ErrorResponse	"Файл удалён"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/applications/{id}/files/{file_id} [get]
func (h *ApplicationHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	appID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
		return
	}
	fileID, err := uuid.Parse(r.PathValue("file_id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "FILE_NOT_FOUND", "Файл не найден", nil)
		return
	}
	role := middleware.UserRoleFromCtx(r.Context())
	url, err := h.appService.DownloadFile(appID, fileID, role == "admin", userID)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// Admin handlers

// AdminList возвращает список всех заявок с фильтрацией и пагинацией.
//
//	@Summary	Список всех заявок (admin)
//	@Tags		admin-applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		status			query		[]string	false	"Фильтр по статусам"	collectionFormat(multi)	Enums(new,in_review,printing,ready,issued,rejected,cancelled)
//	@Param		search			query		string		false	"Поиск по имени / email"
//	@Param		material_id		query		string		false	"UUID материала"	format(uuid)
//	@Param		created_from	query		string		false	"Дата создания от (YYYY-MM-DD)"
//	@Param		created_to		query		string		false	"Дата создания до (YYYY-MM-DD)"
//	@Param		desired_from	query		string		false	"Желаемая дата от (YYYY-MM-DD)"
//	@Param		desired_to		query		string		false	"Желаемая дата до (YYYY-MM-DD)"
//	@Param		page			query		int			false	"Страница"				default(1)
//	@Param		per_page		query		int			false	"Записей на странице"	default(20)	maximum(100)
//	@Success	200	{object}	dto.AdminApplicationListResponse
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/admin/applications [get]
func (h *ApplicationHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, perPage := parsePagination(r)

	filter := dto.ApplicationFilter{
		Statuses: q["status"],
		Page:     page,
		PerPage:  perPage,
		Search:   q.Get("search"),
	}

	if midStr := q.Get("material_id"); midStr != "" {
		mid, err := uuid.Parse(midStr)
		if err == nil {
			filter.MaterialID = &mid
		}
	}
	if v := q.Get("created_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.CreatedFrom = &t
		}
	}
	if v := q.Get("created_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filter.CreatedTo = &end
		}
	}
	if v := q.Get("desired_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.DesiredFrom = &t
		}
	}
	if v := q.Get("desired_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.DesiredTo = &t
		}
	}

	resp, err := h.appService.AdminList(filter)
	if err != nil {
		h.logger.Error("admin list applications", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// AdminGet возвращает детали заявки для администратора.
//
//	@Summary	Детали заявки (admin)
//	@Tags		admin-applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"ID заявки"	format(uuid)
//	@Success	200	{object}	dto.ApplicationDetailAdmin
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse
//	@Failure	404	{object}	apierr.ErrorResponse
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/admin/applications/{id} [get]
func (h *ApplicationHandler) AdminGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
		return
	}
	resp, err := h.appService.AdminGetByID(id)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// AdminChangeStatus изменяет статус заявки.
//
//	@Summary	Изменить статус заявки (admin)
//	@Tags		admin-applications
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string					true	"ID заявки"	format(uuid)
//	@Param		request	body		dto.ChangeStatusRequest	true	"Новый статус"
//	@Success	200		{object}	dto.ChangeStatusResponse
//	@Failure	400		{object}	apierr.ErrorResponse	"Невалидный статус / отсутствует обязательное поле"
//	@Failure	401		{object}	apierr.ErrorResponse
//	@Failure	403		{object}	apierr.ErrorResponse
//	@Failure	404		{object}	apierr.ErrorResponse
//	@Failure	409		{object}	apierr.ErrorResponse	"Заявка в финальном статусе"
//	@Failure	500		{object}	apierr.ErrorResponse
//	@Router		/api/admin/applications/{id}/status [patch]
func (h *ApplicationHandler) AdminChangeStatus(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.UserIDFromCtx(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
		return
	}

	var req dto.ChangeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное тело запроса", nil)
		return
	}

	resp, err := h.appService.AdminChangeStatus(id, adminID, req.Status, req.Comment, req.RejectionReason)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// AdminDownloadFile перенаправляет на presigned URL файла заявки (admin).
//
//	@Summary	Скачать файл заявки (admin)
//	@Tags		admin-applications
//	@Security	BearerAuth
//	@Param		id		path	string	true	"ID заявки"	format(uuid)
//	@Param		file_id	path	string	true	"ID файла"	format(uuid)
//	@Success	302
//	@Failure	401	{object}	apierr.ErrorResponse
//	@Failure	403	{object}	apierr.ErrorResponse
//	@Failure	404	{object}	apierr.ErrorResponse
//	@Failure	410	{object}	apierr.ErrorResponse	"Файл удалён"
//	@Failure	500	{object}	apierr.ErrorResponse
//	@Router		/api/admin/applications/{id}/files/{file_id} [get]
func (h *ApplicationHandler) AdminDownloadFile(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
		return
	}
	fileID, err := uuid.Parse(r.PathValue("file_id"))
	if err != nil {
		apierr.Write(w, http.StatusNotFound, "FILE_NOT_FOUND", "Файл не найден", nil)
		return
	}
	url, err := h.appService.DownloadFileAdmin(appID, fileID)
	if err != nil {
		h.handleAppError(w, err)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *ApplicationHandler) handleAppError(w http.ResponseWriter, err error) {
	var eNotFound *services.ErrApplicationNotFound
	var eFinalized *services.ErrApplicationFinalized
	var eCancel *services.ErrCancelNotAllowed
	var eFilesReq *services.ErrFilesRequired
	var eFormat *services.ErrInvalidFileFormat
	var eTooLarge *services.ErrFileTooLarge
	var eStorage *services.ErrStorageError
	var eProfile *services.ErrProfileNotFound
	var eIncomplete *services.ErrProfileIncomplete
	var eLimit *services.ErrActiveLimitReached
	var eDatePast *services.ErrDesiredDateInPast
	var eMatNotFound *services.ErrMaterialNotFound
	var eMatNA *services.ErrMaterialNotAvailable
	var eColorReq *services.ErrColorRequired
	var eColorNF *services.ErrColorNotFound
	var eColorNA *services.ErrColorNotAvailable
	var eStatusNA *services.ErrStatusNotAllowed
	var eRejReason *services.ErrRejectionReasonRequired
	var eComment *services.ErrCommentRequired
	var eFileLocked *services.ErrFilesLocked
	var eFileNF *services.ErrFileNotFound
	var eFileDeleted *services.ErrFileDeleted
	var eFilesLimit *services.ErrFilesLimitReached

	switch {
	case errors.As(err, &eNotFound):
		apierr.Write(w, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Заявка не найдена", nil)
	case errors.As(err, &eFinalized):
		apierr.Write(w, http.StatusConflict, "APPLICATION_FINALIZED", "Заявка в финальном статусе", nil)
	case errors.As(err, &eCancel):
		apierr.Write(w, http.StatusConflict, "CANCEL_NOT_ALLOWED",
			"Отмена невозможна: заявка уже не в статусе «Новая». Свяжитесь с администратором.",
			map[string]string{"current_status": eCancel.CurrentStatus})
	case errors.As(err, &eFilesReq):
		apierr.Write(w, http.StatusBadRequest, "FILES_REQUIRED", "Необходимо приложить хотя бы один файл", nil)
	case errors.As(err, &eFormat):
		apierr.Write(w, http.StatusBadRequest, "INVALID_FILE_FORMAT",
			"Недопустимый формат файла (ожидается STL, STEP или 3MF)",
			map[string]string{"filename": eFormat.Filename})
	case errors.As(err, &eTooLarge):
		apierr.Write(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			"Файл превышает допустимый размер",
			map[string]any{"filename": eTooLarge.Filename, "max_size_mb": 20})
	case errors.As(err, &eStorage):
		apierr.Write(w, http.StatusBadGateway, "STORAGE_ERROR", "Хранилище файлов недоступно", nil)
	case errors.As(err, &eProfile):
		apierr.Write(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "Профиль не найден", nil)
	case errors.As(err, &eIncomplete):
		apierr.Write(w, http.StatusConflict, "PROFILE_INCOMPLETE",
			"Заполните профиль перед подачей заявки",
			map[string][]string{"missing": eIncomplete.Missing})
	case errors.As(err, &eLimit):
		apierr.Write(w, http.StatusConflict, "ACTIVE_LIMIT_REACHED",
			"Достигнут лимит активных заявок",
			map[string]int{"limit": 10})
	case errors.As(err, &eDatePast):
		apierr.Write(w, http.StatusBadRequest, "DESIRED_DATE_IN_PAST", "Желаемая дата не может быть в прошлом", nil)
	case errors.As(err, &eMatNotFound):
		apierr.Write(w, http.StatusNotFound, "MATERIAL_NOT_FOUND", "Материал не найден", nil)
	case errors.As(err, &eMatNA):
		apierr.Write(w, http.StatusConflict, "MATERIAL_NOT_AVAILABLE", "Материал недоступен", nil)
	case errors.As(err, &eColorReq):
		apierr.Write(w, http.StatusBadRequest, "COLOR_REQUIRED", "Необходимо указать цвет", nil)
	case errors.As(err, &eColorNF):
		apierr.Write(w, http.StatusNotFound, "COLOR_NOT_FOUND", "Цвет не найден", nil)
	case errors.As(err, &eColorNA):
		apierr.Write(w, http.StatusConflict, "COLOR_NOT_AVAILABLE", "Цвет недоступен", nil)
	case errors.As(err, &eStatusNA):
		apierr.Write(w, http.StatusBadRequest, "STATUS_NOT_ALLOWED",
			"Статус недоступен для этой операции",
			map[string][]string{"allowed": {"in_review", "printing", "ready", "issued", "rejected"}})
	case errors.As(err, &eRejReason):
		apierr.Write(w, http.StatusBadRequest, "REJECTION_REASON_REQUIRED", "Укажите причину отклонения", nil)
	case errors.As(err, &eComment):
		apierr.Write(w, http.StatusBadRequest, "COMMENT_REQUIRED", "Добавьте комментарий", nil)
	case errors.As(err, &eFileLocked):
		apierr.Write(w, http.StatusConflict, "FILES_LOCKED",
			"Добавление файлов недоступно: заявка уже не в статусе «Новая»",
			map[string]string{"current_status": eFileLocked.CurrentStatus})
	case errors.As(err, &eFileNF):
		apierr.Write(w, http.StatusNotFound, "FILE_NOT_FOUND", "Файл не найден", nil)
	case errors.As(err, &eFileDeleted):
		apierr.Write(w, http.StatusGone, "FILE_DELETED",
			"Файл удалён по истечении срока хранения",
			map[string]string{"deleted_after": eFileDeleted.DeletedAfter.Format("2006-01-02T15:04:05Z")})
	case errors.As(err, &eFilesLimit):
		apierr.Write(w, http.StatusConflict, "FILES_LIMIT_REACHED",
			"Достигнут лимит файлов для заявки",
			map[string]int{"limit": 10})
	default:
		h.logger.Error("application handler error", "err", err)
		apierr.Write(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
	}
}

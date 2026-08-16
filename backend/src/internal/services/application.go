package services

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/dto"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/models"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/repository"
)

const (
	maxFileSize     = 20 * 1024 * 1024
	maxFilesPerApp  = 10
	maxActiveApps   = 10
	maxArchiveFiles = 100
	maxArchiveSize  = 200 * 1024 * 1024
	cancelFileTTL   = 7 * 24 * time.Hour
	rejectedFileTTL = 7 * 24 * time.Hour
	issuedFileTTL   = 30 * 24 * time.Hour
)

var adminAllowedStatuses = map[string]bool{
	"in_review": true,
	"printing":  true,
	"ready":     true,
	"issued":    true,
	"rejected":  true,
}

type ParsedFile struct {
	Header *multipart.FileHeader
	Format string
}

type FileDownload struct {
	Reader   io.ReadCloser
	Filename string
	Size     int
}

type ApplicationService struct {
	logger          *slog.Logger
	txMgr           repository.TxManager
	appRepo         repository.ApplicationRepository
	fileRepo        repository.FileRepository
	historyRepo     repository.StatusHistoryRepository
	userRepo        repository.UserRepository
	materialService *MaterialService
	storageService  *StorageService
	emailService    *EmailService
}

func NewApplicationService(
	logger *slog.Logger,
	txMgr repository.TxManager,
	appRepo repository.ApplicationRepository,
	fileRepo repository.FileRepository,
	historyRepo repository.StatusHistoryRepository,
	userRepo repository.UserRepository,
	materialService *MaterialService,
	storageService *StorageService,
	emailService *EmailService,
) *ApplicationService {
	return &ApplicationService{
		logger:          logger,
		txMgr:           txMgr,
		appRepo:         appRepo,
		fileRepo:        fileRepo,
		historyRepo:     historyRepo,
		userRepo:        userRepo,
		materialService: materialService,
		storageService:  storageService,
		emailService:    emailService,
	}
}

func detectFormat(filename string, data []byte) (string, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".stl":
		// Binary STL: 80-byte header + 4-byte triangle count = 84 bytes minimum.
		// ASCII STL starts with "solid" and is always longer.
		if len(data) < 84 {
			return "", false
		}
		return "STL", true
	case ".step", ".stp":
		if len(data) >= 12 && strings.HasPrefix(string(data[:12]), "ISO-10303-21") {
			return "STEP", true
		}
		return "", false
	case ".3mf":
		if len(data) >= 2 && data[0] == 0x50 && data[1] == 0x4B {
			return "3MF", true
		}
		return "", false
	case ".zip":
		if len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B {
			return "ZIP", true
		}
		return "", false
	}
	return "", false
}

func validateZIP(file multipart.File, size int64) bool {
	archive, err := zip.NewReader(file, size)
	if err != nil || len(archive.File) == 0 || len(archive.File) > maxArchiveFiles {
		return false
	}

	var totalSize uint64
	hasModel := false
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		cleanName := path.Clean(strings.ReplaceAll(entry.Name, "\\", "/"))
		if path.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
			return false
		}
		if entry.UncompressedSize64 > maxArchiveSize || totalSize > uint64(maxArchiveSize)-entry.UncompressedSize64 {
			return false
		}
		totalSize += entry.UncompressedSize64
		switch strings.ToLower(filepath.Ext(cleanName)) {
		case ".stl", ".step", ".stp", ".3mf":
			reader, err := entry.Open()
			if err != nil {
				return false
			}
			headerSize := uint64(84)
			if entry.UncompressedSize64 < headerSize {
				headerSize = entry.UncompressedSize64
			}
			header := make([]byte, headerSize)
			_, readErr := io.ReadFull(reader, header)
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil {
				return false
			}
			if _, ok := detectFormat(cleanName, header); !ok {
				return false
			}
			hasModel = true
		}
	}
	return hasModel
}

func validateFile(fh *multipart.FileHeader) (string, error) {
	if fh.Size <= 0 {
		return "", &ErrInvalidFileFormat{Filename: fh.Filename}
	}
	if fh.Size > maxFileSize {
		return "", &ErrFileTooLarge{Filename: fh.Filename}
	}

	file, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	headerSize := int64(84)
	if fh.Size < headerSize {
		headerSize = fh.Size
	}
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(file, header); err != nil {
		return "", fmt.Errorf("read file header: %w", err)
	}
	format, ok := detectFormat(fh.Filename, header)
	if !ok {
		return "", &ErrInvalidFileFormat{Filename: fh.Filename}
	}
	if format == "ZIP" && !validateZIP(file, fh.Size) {
		return "", &ErrInvalidFileFormat{Filename: fh.Filename}
	}
	return format, nil
}

func (s *ApplicationService) validateFiles(files []*multipart.FileHeader) ([]ParsedFile, error) {
	if len(files) == 0 {
		return nil, &ErrFilesRequired{}
	}
	if len(files) > maxFilesPerApp {
		return nil, &ErrFilesLimitReached{}
	}
	parsed := make([]ParsedFile, 0, len(files))
	for _, fh := range files {
		format, err := validateFile(fh)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, ParsedFile{Header: fh, Format: format})
	}
	return parsed, nil
}

func (s *ApplicationService) uploadFile(ctx context.Context, storagePath string, file *multipart.FileHeader) error {
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open file for upload: %w", err)
	}
	defer reader.Close()
	return s.storageService.Upload(ctx, storagePath, reader, file.Size, "application/octet-stream")
}

func (s *ApplicationService) ListByUser(userID uuid.UUID, status string, page, perPage int) (*dto.ApplicationListResponse, error) {
	apps, total, err := s.appRepo.ListByUser(userID, status, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}

	appIDs := make([]uuid.UUID, len(apps))
	for i, a := range apps {
		appIDs[i] = a.ID
	}
	fileCounts, err := s.fileRepo.CountsByApplicationIDs(appIDs)
	if err != nil {
		return nil, fmt.Errorf("count files: %w", err)
	}

	items := make([]*dto.ApplicationListItem, 0, len(apps))
	for _, a := range apps {
		items = append(items, &dto.ApplicationListItem{
			ID:           a.ID.String(),
			Number:       a.Number,
			Title:        a.Title,
			Status:       a.Status,
			MaterialName: a.SnapshotMaterialName,
			ColorName:    a.SnapshotColorName,
			DesiredDate:  a.DesiredDate.Format("2006-01-02"),
			CreatedAt:    a.CreatedAt,
			FilesCount:   fileCounts[a.ID],
		})
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}
	return &dto.ApplicationListResponse{
		Items: items,
		Meta:  dto.PaginationMeta{Page: page, PerPage: perPage, Total: total, TotalPages: totalPages},
	}, nil
}

func (s *ApplicationService) Create(
	ctx context.Context,
	userID uuid.UUID,
	title string,
	position, purpose string,
	materialID uuid.UUID,
	colorMatters bool,
	colorID *uuid.UUID,
	desiredDate time.Time,
	comment *string,
	fileURL *string,
	files []*multipart.FileHeader,
) (*dto.CreateApplicationResponse, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 255 || strings.ContainsAny(title, "\r\n") {
		return nil, &ErrInvalidApplicationTitle{}
	}

	if fileURL != nil {
		trimmed := strings.TrimSpace(*fileURL)
		parsedURL, err := url.ParseRequestURI(trimmed)
		if err != nil || len(trimmed) > 2048 || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return nil, &ErrInvalidFileURL{}
		}
		fileURL = &trimmed
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, &ErrProfileNotFound{}
	}
	var missing []string
	if user.FullName == nil {
		missing = append(missing, "full_name")
	}
	if user.Telegram == nil && user.Max == nil {
		missing = append(missing, "telegram", "max")
	}
	if len(missing) > 0 {
		return nil, &ErrProfileIncomplete{Missing: missing}
	}

	today := time.Now().Truncate(24 * time.Hour)
	if desiredDate.Before(today) {
		return nil, &ErrDesiredDateInPast{}
	}

	matName, colorName, err := s.materialService.GetMaterialForApplication(materialID, colorMatters, colorID)
	if err != nil {
		return nil, err
	}

	var parsedFiles []ParsedFile
	if len(files) > 0 {
		parsedFiles, err = s.validateFiles(files)
		if err != nil {
			return nil, err
		}
	} else if fileURL == nil {
		return nil, &ErrFilesRequired{}
	}

	year := time.Now().Year()
	number, err := s.appRepo.GenerateNumber(year)
	if err != nil {
		return nil, fmt.Errorf("generate number: %w", err)
	}

	appModel := &models.Application{
		Number:               number,
		Title:                title,
		UserID:               userID,
		SnapshotFullName:     *user.FullName,
		SnapshotEmail:        user.Email,
		Position:             position,
		Purpose:              purpose,
		MaterialID:           materialID,
		SnapshotMaterialName: matName,
		ColorMatters:         colorMatters,
		SnapshotColorName:    colorName,
		DesiredDate:          desiredDate,
		Comment:              comment,
		FileURL:              fileURL,
	}
	if colorMatters && colorID != nil {
		appModel.ColorID = colorID
	}

	// Track paths of files uploaded to MinIO so we can clean up on any failure.
	uploadedPaths := make([]string, 0, len(parsedFiles))
	cleanupUploads := func() {
		for _, p := range uploadedPaths {
			if delErr := s.storageService.Delete(context.Background(), p); delErr != nil {
				s.logger.Error("cleanup orphan file", "path", p, "err", delErr)
			}
		}
	}

	var app *models.Application
	txErr := s.txMgr.RunInTx(ctx, func(tx repository.DBTX) error {
		activeCount, err := s.appRepo.CountActive(tx, userID)
		if err != nil {
			return fmt.Errorf("count active: %w", err)
		}
		if activeCount >= maxActiveApps {
			return &ErrActiveLimitReached{}
		}

		app, err = s.appRepo.Create(tx, appModel)
		if err != nil {
			return fmt.Errorf("create application: %w", err)
		}

		_, err = s.historyRepo.Create(tx, &models.StatusHistory{
			ApplicationID: app.ID,
			Status:        "new",
			ChangedBy:     userID,
		})
		if err != nil {
			return fmt.Errorf("create status history: %w", err)
		}

		for _, pf := range parsedFiles {
			fileID := uuid.New()
			ext := strings.ToLower(filepath.Ext(pf.Header.Filename))
			storagePath := fmt.Sprintf("applications/%s/%s%s", app.ID, fileID, ext)

			if err := s.uploadFile(ctx, storagePath, pf.Header); err != nil {
				cleanupUploads()
				s.logger.Error("minio upload failed", "err", err)
				return &ErrStorageError{}
			}
			uploadedPaths = append(uploadedPaths, storagePath)

			if _, err := s.fileRepo.Create(tx, &models.File{
				ApplicationID: app.ID,
				Filename:      pf.Header.Filename,
				StoragePath:   storagePath,
				Size:          int(pf.Header.Size),
				Format:        pf.Format,
			}); err != nil {
				cleanupUploads()
				return fmt.Errorf("create file record: %w", err)
			}
		}
		return nil
	})

	if txErr != nil {
		// Clean up any files that were uploaded before the tx commit failed.
		cleanupUploads()
		return nil, txErr
	}

	if err := s.emailService.SendApplicationCreated(user.Email, number, title, app.ID.String()); err != nil {
		s.logger.Error("send application created email", "err", err, "number", number)
	}

	adminRecipients, err := s.userRepo.ListApplicationNotificationRecipients()
	if err != nil {
		s.logger.Error("list application notification recipients", "err", err, "number", number)
	} else {
		for _, recipient := range adminRecipients {
			if err := s.emailService.SendNewApplicationToAdmin(
				recipient.Email,
				number,
				title,
				app.ID.String(),
				*user.FullName,
				user.Email,
				matName,
				desiredDate.Format("02.01.2006"),
				purpose,
			); err != nil {
				s.logger.Error(
					"send new application email to admin",
					"err", err,
					"number", number,
					"admin_id", recipient.ID,
				)
			}
		}
	}

	return &dto.CreateApplicationResponse{
		ID:        app.ID.String(),
		Number:    app.Number,
		Title:     app.Title,
		Status:    app.Status,
		CreatedAt: app.CreatedAt,
	}, nil
}

func (s *ApplicationService) GetByIDAndUser(id, userID uuid.UUID) (*dto.ApplicationDetailUser, error) {
	app, err := s.appRepo.FindByIDAndUser(id, userID)
	if err != nil {
		return nil, fmt.Errorf("find application: %w", err)
	}
	if app == nil {
		return nil, &ErrApplicationNotFound{}
	}
	return s.buildUserDetail(app)
}

func (s *ApplicationService) buildUserDetail(app *models.Application) (*dto.ApplicationDetailUser, error) {
	files, err := s.fileRepo.ListByApplication(app.ID)
	if err != nil {
		return nil, err
	}
	history, err := s.historyRepo.ListByApplication(app.ID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, 0, len(history))
	seen := make(map[uuid.UUID]bool)
	for _, h := range history {
		if !seen[h.ChangedBy] {
			userIDs = append(userIDs, h.ChangedBy)
			seen[h.ChangedBy] = true
		}
	}
	userMap, err := s.userRepo.FindByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	fileItems := make([]*dto.FileItem, 0, len(files))
	for _, f := range files {
		fileItems = append(fileItems, &dto.FileItem{
			ID:       f.ID.String(),
			Filename: f.Filename,
			Size:     f.Size,
			Format:   f.Format,
		})
	}

	histItems := make([]*dto.StatusHistoryItemUser, 0, len(history))
	for _, h := range history {
		role := "user"
		if u, ok := userMap[h.ChangedBy]; ok {
			role = u.Role
		}
		histItems = append(histItems, &dto.StatusHistoryItemUser{
			Status:    h.Status,
			Comment:   h.Comment,
			ChangedBy: role,
			CreatedAt: h.CreatedAt,
		})
	}

	detail := &dto.ApplicationDetailUser{
		ID:               app.ID.String(),
		Number:           app.Number,
		Title:            app.Title,
		Status:           app.Status,
		RejectionReason:  app.RejectionReason,
		Position:         app.Position,
		Purpose:          app.Purpose,
		Material:         dto.MaterialRef{ID: app.MaterialID.String(), SnapshotName: app.SnapshotMaterialName},
		ColorMatters:     app.ColorMatters,
		DesiredDate:      app.DesiredDate.Format("2006-01-02"),
		Comment:          app.Comment,
		FileURL:          app.FileURL,
		Files:            fileItems,
		FilesDeleteAfter: app.FilesDeleteAfter,
		StatusHistory:    histItems,
		CreatedAt:        app.CreatedAt,
	}
	if app.ColorID != nil && app.SnapshotColorName != nil {
		detail.Color = &dto.ColorRef{ID: app.ColorID.String(), SnapshotName: *app.SnapshotColorName}
	}
	return detail, nil
}

func (s *ApplicationService) CancelByUser(id, userID uuid.UUID) (*dto.CancelApplicationResponse, error) {
	filesDeleteAfter := time.Now().Add(cancelFileTTL)

	var app *models.Application
	txErr := s.txMgr.RunInTx(context.Background(), func(tx repository.DBTX) error {
		var err error
		app, err = s.appRepo.CancelAtomic(tx, id, userID, filesDeleteAfter)
		if err != nil {
			return fmt.Errorf("cancel application: %w", err)
		}
		if app == nil {
			return nil // resolved below
		}
		_, err = s.historyRepo.Create(tx, &models.StatusHistory{
			ApplicationID: app.ID,
			Status:        "cancelled",
			ChangedBy:     userID,
		})
		return err
	})
	if txErr != nil {
		return nil, txErr
	}
	if app == nil {
		existing, _ := s.appRepo.FindByIDAndUser(id, userID)
		if existing == nil {
			return nil, &ErrApplicationNotFound{}
		}
		return nil, &ErrCancelNotAllowed{CurrentStatus: existing.Status}
	}

	return &dto.CancelApplicationResponse{
		ID:               app.ID.String(),
		Status:           app.Status,
		FilesDeleteAfter: app.FilesDeleteAfter,
	}, nil
}

// Admin methods

func (s *ApplicationService) AdminList(filter dto.ApplicationFilter) (*dto.AdminApplicationListResponse, error) {
	apps, total, err := s.appRepo.ListAdmin(filter)
	if err != nil {
		return nil, fmt.Errorf("list admin applications: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)
	items := make([]*dto.AdminApplicationListItem, 0, len(apps))
	for _, a := range apps {
		deadlineSoon := a.Status == "new" && (a.DesiredDate.Before(today.AddDate(0, 0, 4)))
		items = append(items, &dto.AdminApplicationListItem{
			ID:           a.ID.String(),
			Number:       a.Number,
			Title:        a.Title,
			FullName:     a.SnapshotFullName,
			CreatedAt:    a.CreatedAt,
			DesiredDate:  a.DesiredDate.Format("2006-01-02"),
			MaterialName: a.SnapshotMaterialName,
			Status:       a.Status,
			DeadlineSoon: deadlineSoon,
		})
	}

	totalPages := (total + filter.PerPage - 1) / filter.PerPage
	if totalPages == 0 {
		totalPages = 1
	}
	return &dto.AdminApplicationListResponse{
		Items: items,
		Meta:  dto.PaginationMeta{Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPages: totalPages},
	}, nil
}

func (s *ApplicationService) AdminGetByID(id uuid.UUID) (*dto.ApplicationDetailAdmin, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find application: %w", err)
	}
	if app == nil {
		return nil, &ErrApplicationNotFound{}
	}

	user, err := s.userRepo.FindByID(app.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	files, err := s.fileRepo.ListByApplication(app.ID)
	if err != nil {
		return nil, err
	}
	history, err := s.historyRepo.ListByApplication(app.ID)
	if err != nil {
		return nil, err
	}

	material, _ := s.materialService.FindByID(app.MaterialID)
	isActiveNow := material != nil && material.IsActive

	userIDs := make([]uuid.UUID, 0, len(history))
	seen := make(map[uuid.UUID]bool)
	for _, h := range history {
		if !seen[h.ChangedBy] {
			userIDs = append(userIDs, h.ChangedBy)
			seen[h.ChangedBy] = true
		}
	}
	userMap, err := s.userRepo.FindByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	fileItems := make([]*dto.FileItem, 0, len(files))
	for _, f := range files {
		fileItems = append(fileItems, &dto.FileItem{
			ID:       f.ID.String(),
			Filename: f.Filename,
			Size:     f.Size,
			Format:   f.Format,
		})
	}

	histItems := make([]*dto.StatusHistoryItemAdmin, 0, len(history))
	for _, h := range history {
		var changedBy *dto.StatusHistoryChangedBy
		if u, ok := userMap[h.ChangedBy]; ok {
			fullName := ""
			if u.FullName != nil {
				fullName = *u.FullName
			}
			changedBy = &dto.StatusHistoryChangedBy{
				ID:       u.ID.String(),
				FullName: fullName,
				Role:     u.Role,
			}
		}
		histItems = append(histItems, &dto.StatusHistoryItemAdmin{
			Status:    h.Status,
			Comment:   h.Comment,
			ChangedBy: changedBy,
			CreatedAt: h.CreatedAt,
		})
	}

	detail := &dto.ApplicationDetailAdmin{
		ID:     app.ID.String(),
		Number: app.Number,
		Title:  app.Title,
		Applicant: dto.AdminApplicant{
			UserID:           user.ID.String(),
			SnapshotFullName: app.SnapshotFullName,
			SnapshotEmail:    app.SnapshotEmail,
			Telegram:         user.Telegram,
			Max:              user.Max,
		},
		Position: app.Position,
		Purpose:  app.Purpose,
		Material: dto.MaterialRefAdmin{
			ID:           app.MaterialID.String(),
			SnapshotName: app.SnapshotMaterialName,
			IsActiveNow:  isActiveNow,
		},
		ColorMatters:    app.ColorMatters,
		DesiredDate:     app.DesiredDate.Format("2006-01-02"),
		Comment:         app.Comment,
		FileURL:         app.FileURL,
		Status:          app.Status,
		RejectionReason: app.RejectionReason,
		Files:           fileItems,
		StatusHistory:   histItems,
		CreatedAt:       app.CreatedAt,
	}
	if app.ColorID != nil && app.SnapshotColorName != nil {
		detail.Color = &dto.ColorRef{ID: app.ColorID.String(), SnapshotName: *app.SnapshotColorName}
	}
	return detail, nil
}

func (s *ApplicationService) AdminChangeStatus(id, adminID uuid.UUID, targetStatus string, comment, rejectionReason *string) (*dto.ChangeStatusResponse, error) {
	if !adminAllowedStatuses[targetStatus] {
		return nil, &ErrStatusNotAllowed{}
	}

	app, err := s.appRepo.FindByID(id)
	if err != nil || app == nil {
		return nil, &ErrApplicationNotFound{}
	}
	if app.Status == "cancelled" {
		return nil, &ErrApplicationFinalized{}
	}
	if targetStatus == "rejected" && (rejectionReason == nil || strings.TrimSpace(*rejectionReason) == "") {
		return nil, &ErrRejectionReasonRequired{}
	}

	if targetStatus == app.Status {
		if comment == nil || strings.TrimSpace(*comment) == "" {
			return nil, &ErrCommentRequired{}
		}
		if err := s.txMgr.RunInTx(context.Background(), func(tx repository.DBTX) error {
			_, err := s.historyRepo.Create(tx, &models.StatusHistory{
				ApplicationID: app.ID,
				Status:        targetStatus,
				Comment:       comment,
				ChangedBy:     adminID,
			})
			return err
		}); err != nil {
			return nil, fmt.Errorf("create history: %w", err)
		}
		return &dto.ChangeStatusResponse{
			ID:               app.ID.String(),
			Status:           app.Status,
			FilesDeleteAfter: app.FilesDeleteAfter,
		}, nil
	}

	var filesDeleteAfter *time.Time
	now := time.Now()
	switch targetStatus {
	case "rejected":
		t := now.Add(rejectedFileTTL)
		filesDeleteAfter = &t
	case "issued":
		t := now.Add(issuedFileTTL)
		filesDeleteAfter = &t
	}

	var updatedApp *models.Application
	if err := s.txMgr.RunInTx(context.Background(), func(tx repository.DBTX) error {
		var err error
		updatedApp, err = s.appRepo.UpdateStatus(tx, id, targetStatus, rejectionReason, filesDeleteAfter)
		if err != nil {
			return fmt.Errorf("update status: %w", err)
		}
		_, err = s.historyRepo.Create(tx, &models.StatusHistory{
			ApplicationID: app.ID,
			Status:        targetStatus,
			Comment:       comment,
			ChangedBy:     adminID,
		})
		return err
	}); err != nil {
		return nil, err
	}

	reason := ""
	if rejectionReason != nil {
		reason = *rejectionReason
	}
	if err := s.emailService.SendStatusChanged(app.SnapshotEmail, app.Number, app.Title, app.ID.String(), targetStatus, reason); err != nil {
		s.logger.Error("send status changed email", "err", err)
	}

	return &dto.ChangeStatusResponse{
		ID:               updatedApp.ID.String(),
		Status:           updatedApp.Status,
		FilesDeleteAfter: updatedApp.FilesDeleteAfter,
	}, nil
}

func (s *ApplicationService) UploadFileToApp(ctx context.Context, appID, userID uuid.UUID, fh *multipart.FileHeader) (*dto.UploadFileResponse, error) {
	app, err := s.appRepo.FindByIDAndUser(appID, userID)
	if err != nil {
		return nil, fmt.Errorf("find application: %w", err)
	}
	if app == nil {
		return nil, &ErrApplicationNotFound{}
	}
	if app.Status != "new" {
		return nil, &ErrFilesLocked{CurrentStatus: app.Status}
	}

	existingCount, err := s.fileRepo.CountByApplication(appID)
	if err != nil {
		return nil, fmt.Errorf("count files: %w", err)
	}
	if existingCount >= maxFilesPerApp {
		return nil, &ErrFilesLimitReached{}
	}

	format, err := validateFile(fh)
	if err != nil {
		return nil, err
	}

	fileID := uuid.New()
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	storagePath := fmt.Sprintf("applications/%s/%s%s", appID, fileID, ext)

	if err := s.uploadFile(ctx, storagePath, fh); err != nil {
		return nil, &ErrStorageError{}
	}

	var created *models.File
	if err := s.txMgr.RunInTx(ctx, func(tx repository.DBTX) error {
		var err error
		created, err = s.fileRepo.Create(tx, &models.File{
			ApplicationID: appID,
			Filename:      fh.Filename,
			StoragePath:   storagePath,
			Size:          int(fh.Size),
			Format:        format,
		})
		return err
	}); err != nil {
		if delErr := s.storageService.Delete(context.Background(), storagePath); delErr != nil {
			s.logger.Error("cleanup orphan file after db error", "path", storagePath, "err", delErr)
		}
		return nil, fmt.Errorf("create file record: %w", err)
	}

	return &dto.UploadFileResponse{
		ID:        created.ID.String(),
		Filename:  created.Filename,
		Size:      created.Size,
		Format:    created.Format,
		CreatedAt: created.CreatedAt,
	}, nil
}

func (s *ApplicationService) DownloadFile(
	ctx context.Context,
	appID, fileID uuid.UUID,
	isAdmin bool,
	userID uuid.UUID,
) (*FileDownload, error) {
	var app *models.Application
	var err error
	if isAdmin {
		app, err = s.appRepo.FindByID(appID)
	} else {
		app, err = s.appRepo.FindByIDAndUser(appID, userID)
	}
	if err != nil || app == nil {
		return nil, &ErrApplicationNotFound{}
	}

	file, err := s.fileRepo.FindByID(fileID, appID)
	if err != nil {
		return nil, fmt.Errorf("find file: %w", err)
	}
	if file == nil {
		return nil, &ErrFileNotFound{}
	}
	if file.DeletedAt != nil {
		return nil, &ErrFileDeleted{DeletedAfter: *file.DeletedAt}
	}

	reader, err := s.storageService.Open(ctx, file.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("open stored file: %w", err)
	}
	return &FileDownload{Reader: reader, Filename: file.Filename, Size: file.Size}, nil
}

func (s *ApplicationService) DownloadFileAdmin(ctx context.Context, appID, fileID uuid.UUID) (*FileDownload, error) {
	return s.DownloadFile(ctx, appID, fileID, true, uuid.Nil)
}

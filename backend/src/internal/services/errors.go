package services

import "time"

type ErrInvalidDomain struct {
	AllowedDomains []string
}

func (e *ErrInvalidDomain) Error() string { return "INVALID_DOMAIN" }

type ErrRateLimitIP struct {
	RetryAfter int
}

func (e *ErrRateLimitIP) Error() string { return "RATE_LIMIT_IP" }

type ErrRateLimitEmail struct {
	RetryAfter int
}

func (e *ErrRateLimitEmail) Error() string { return "RATE_LIMIT_EMAIL" }

type ErrEmailProvider struct{}

func (e *ErrEmailProvider) Error() string { return "EMAIL_PROVIDER_ERROR" }

type ErrOTPNotFound struct{}

func (e *ErrOTPNotFound) Error() string { return "OTP_NOT_FOUND" }

type ErrOTPLocked struct {
	LockedUntil time.Time
}

func (e *ErrOTPLocked) Error() string { return "LOCKED" }

type ErrOTPExpired struct{}

func (e *ErrOTPExpired) Error() string { return "CODE_EXPIRED" }

type ErrInvalidCode struct {
	AttemptsLeft int
}

func (e *ErrInvalidCode) Error() string { return "INVALID_CODE" }

type ErrInvalidRefreshToken struct{}

func (e *ErrInvalidRefreshToken) Error() string { return "INVALID_REFRESH_TOKEN" }

type ErrRefreshTokenExpired struct{}

func (e *ErrRefreshTokenExpired) Error() string { return "REFRESH_TOKEN_EXPIRED" }

type ErrRefreshTokenRevoked struct{}

func (e *ErrRefreshTokenRevoked) Error() string { return "REFRESH_TOKEN_REVOKED" }

type ErrRefreshTokenReused struct{}

func (e *ErrRefreshTokenReused) Error() string { return "REFRESH_TOKEN_REUSED" }

// Profile errors

type ErrProfileNotFound struct{}

func (e *ErrProfileNotFound) Error() string { return "PROFILE_NOT_FOUND" }

type ErrContactRequired struct{}

func (e *ErrContactRequired) Error() string { return "CONTACT_REQUIRED" }

type ErrFieldNotEditable struct {
	Field string
}

func (e *ErrFieldNotEditable) Error() string { return "FIELD_NOT_EDITABLE" }

type ErrInvalidFullName struct{}

func (e *ErrInvalidFullName) Error() string { return "INVALID_FULL_NAME" }

// Application errors

type ErrProfileIncomplete struct {
	Missing []string
}

func (e *ErrProfileIncomplete) Error() string { return "PROFILE_INCOMPLETE" }

type ErrActiveLimitReached struct{}

func (e *ErrActiveLimitReached) Error() string { return "ACTIVE_LIMIT_REACHED" }

type ErrDesiredDateInPast struct{}

func (e *ErrDesiredDateInPast) Error() string { return "DESIRED_DATE_IN_PAST" }

type ErrMaterialNotFound struct{}

func (e *ErrMaterialNotFound) Error() string { return "MATERIAL_NOT_FOUND" }

type ErrMaterialNotAvailable struct{}

func (e *ErrMaterialNotAvailable) Error() string { return "MATERIAL_NOT_AVAILABLE" }

type ErrMaterialNameExists struct {
	ExistingID string
}

func (e *ErrMaterialNameExists) Error() string { return "MATERIAL_NAME_EXISTS" }

type ErrColorRequired struct{}

func (e *ErrColorRequired) Error() string { return "COLOR_REQUIRED" }

type ErrColorNotFound struct{}

func (e *ErrColorNotFound) Error() string { return "COLOR_NOT_FOUND" }

type ErrColorNotAvailable struct{}

func (e *ErrColorNotAvailable) Error() string { return "COLOR_NOT_AVAILABLE" }

type ErrColorNameExists struct{}

func (e *ErrColorNameExists) Error() string { return "COLOR_NAME_EXISTS" }

type ErrApplicationNotFound struct{}

func (e *ErrApplicationNotFound) Error() string { return "APPLICATION_NOT_FOUND" }

type ErrApplicationFinalized struct{}

func (e *ErrApplicationFinalized) Error() string { return "APPLICATION_FINALIZED" }

type ErrCancelNotAllowed struct {
	CurrentStatus string
}

func (e *ErrCancelNotAllowed) Error() string { return "CANCEL_NOT_ALLOWED" }

type ErrFilesRequired struct{}

func (e *ErrFilesRequired) Error() string { return "FILES_REQUIRED" }

type ErrInvalidFileFormat struct {
	Filename string
}

func (e *ErrInvalidFileFormat) Error() string { return "INVALID_FILE_FORMAT" }

type ErrFileTooLarge struct {
	Filename string
}

func (e *ErrFileTooLarge) Error() string { return "FILE_TOO_LARGE" }

type ErrStorageError struct{}

func (e *ErrStorageError) Error() string { return "STORAGE_ERROR" }

type ErrStatusNotAllowed struct{}

func (e *ErrStatusNotAllowed) Error() string { return "STATUS_NOT_ALLOWED" }

type ErrRejectionReasonRequired struct{}

func (e *ErrRejectionReasonRequired) Error() string { return "REJECTION_REASON_REQUIRED" }

type ErrCommentRequired struct{}

func (e *ErrCommentRequired) Error() string { return "COMMENT_REQUIRED" }

type ErrFilesLocked struct {
	CurrentStatus string
}

func (e *ErrFilesLocked) Error() string { return "FILES_LOCKED" }

type ErrFilesLimitReached struct{}

func (e *ErrFilesLimitReached) Error() string { return "FILES_LIMIT_REACHED" }

type ErrFileNotFound struct{}

func (e *ErrFileNotFound) Error() string { return "FILE_NOT_FOUND" }

type ErrFileDeleted struct {
	DeletedAfter time.Time
}

func (e *ErrFileDeleted) Error() string { return "FILE_DELETED" }

// Admin errors

type ErrUserNotFound struct{}

func (e *ErrUserNotFound) Error() string { return "USER_NOT_FOUND" }

type ErrInvalidRole struct{}

func (e *ErrInvalidRole) Error() string { return "INVALID_ROLE" }

type ErrLastAdmin struct{}

func (e *ErrLastAdmin) Error() string { return "LAST_ADMIN" }

type ErrQueryTooShort struct{}

func (e *ErrQueryTooShort) Error() string { return "QUERY_TOO_SHORT" }

type ErrInvalidPeriod struct{}

func (e *ErrInvalidPeriod) Error() string { return "INVALID_PERIOD" }

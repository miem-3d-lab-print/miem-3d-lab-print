package dto

import "time"

type ProfileResponse struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	FullName     *string    `json:"full_name"`
	Telegram     *string    `json:"telegram"`
	Max          *string    `json:"max"`
	Role         string     `json:"role"`
	ConsentGiven bool       `json:"consent_given"`
	ConsentDate  *time.Time `json:"consent_date"`
	CreatedAt    time.Time  `json:"created_at"`
}

type PatchProfileRequest struct {
	FullName *string `json:"full_name"`
	Telegram *string `json:"telegram"`
	Max      *string `json:"max"`
}

type ConsentResponse struct {
	ConsentGiven bool       `json:"consent_given"`
	ConsentDate  *time.Time `json:"consent_date"`
}

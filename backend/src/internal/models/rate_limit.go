package models

import "time"

const (
	RateLimitTypeEmail = "email"
	RateLimitTypeIP    = "ip"
)

type RateLimit struct {
	ID           int64     `db:"id" json:"id"`
	LimitType    string    `db:"limit_type" json:"limit_type"`
	KeyValue     string    `db:"key_value" json:"key_value"`
	WindowStart  time.Time `db:"window_start" json:"window_start"`
	RequestCount int16     `db:"request_count" json:"request_count"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

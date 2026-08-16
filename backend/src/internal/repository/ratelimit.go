package repository

import "time"

type RateLimitRepository interface {
	GetRequests(limitType, key string, since time.Time) ([]time.Time, error)
	Record(limitType, key string) error
}

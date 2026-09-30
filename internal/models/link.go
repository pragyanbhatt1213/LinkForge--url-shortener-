package models

import "time"

type Link struct {
	ID          int
	Slug        string
	OriginalURL string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

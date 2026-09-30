package models

import "time"

type ClickEvent struct {
	Slug      string
	ClickedAt time.Time
}

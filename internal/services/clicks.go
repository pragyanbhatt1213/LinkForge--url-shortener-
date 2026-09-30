package services

import (
	"time"

	"linkforge-url-shortener/internal/models"
)

var ClickQueue = make(chan models.ClickEvent, 100)

func RecordClick(slug string) {
	ClickQueue <- models.ClickEvent{
		Slug:      slug,
		ClickedAt: time.Now(),
	}
}

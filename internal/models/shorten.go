package models

type ShortenRequest struct {
	URL         string `json:"url"`
	CustomSlug  string `json:"custom_slug"`
	ExpiryHours int    `json:"expiry_hours"`
}

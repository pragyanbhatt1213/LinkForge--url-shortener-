package services

import (
	"errors"
	"linkforge-url-shortener/internal/store"
	"net/url"
	"time"
)

func CreateShortURL(rawURL string, customSlug string, expiryHours int) (string, error) {
	err := ValidateURL(rawURL)
	if err != nil {
		return "", err
	}

	// Choose the slug
	slug := customSlug

	if slug == "" {
		slug = GenerateSlug()
	}
	if customSlug != "" {
		if len(customSlug) < 3 || len(customSlug) > 50 {
			return "", errors.New("custom slug must be between 3 and 50 characters")
		}
	}
	// Default expiry = 24 hours
	expiry := 24 * time.Hour

	// If user provided expiry, use it
	if expiryHours < 0 {
		return "", errors.New("expiry must be greater than 0")
	}

	if expiryHours > 0 {
		expiry = time.Duration(expiryHours) * time.Hour
	}

	// Save URL in Redis with the same expiry
	saved, err := store.SaveSlug(slug, rawURL, expiry)
	if err != nil {
		return "", err
	}

	// Slug already exists
	if !saved {
		return "", errors.New("slug already exists")
	}

	// Calculate PostgreSQL expiry time
	expiryTime := time.Now().Add(expiry)

	// Save permanent record in PostgreSQL
	err = store.SaveLinkWithExpiry(slug, rawURL, &expiryTime)
	if err != nil {
		return "", err
	}

	return slug, nil
}
func ValidateURL(rawURL string) error {
	parsedURL, err := url.ParseRequestURI(rawURL)

	if err != nil {
		return errors.New("invalid URL")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must use http or https")
	}

	if parsedURL.Host == "" {
		return errors.New("URL must have a host")
	}

	return nil
}

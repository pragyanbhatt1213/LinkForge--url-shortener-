package store

import (
	"context"
	"time"

	"linkforge-url-shortener/internal/config"
	"linkforge-url-shortener/internal/models"

	"github.com/jackc/pgx/v5"
)

var postgresDB *pgx.Conn

func NewPostgres() (*pgx.Conn, error) {
	cfg := config.NewPostgresConfig()

	return pgx.ConnectConfig(
		context.Background(),
		cfg,
	)
}

func SetPostgresDB(db *pgx.Conn) {
	postgresDB = db
}

func SaveLink(slug string, originalURL string) error {
	query := `
		INSERT INTO links (slug, original_url)
		VALUES ($1, $2)
	`

	_, err := postgresDB.Exec(
		context.Background(),
		query,
		slug,
		originalURL,
	)

	return err
}

func GetLink(slug string) (string, error) {
	var originalURL string

	query := `
		SELECT original_url
		FROM links
		WHERE slug = $1
	`

	err := postgresDB.QueryRow(
		context.Background(),
		query,
		slug,
	).Scan(&originalURL)

	return originalURL, err
}

func SaveClick(event models.ClickEvent) error {
	query := `
		INSERT INTO clicks (slug, clicked_at)
		VALUES ($1, $2)
	`

	_, err := postgresDB.Exec(
		context.Background(),
		query,
		event.Slug,
		event.ClickedAt,
	)

	return err
}

func GetClickCount(slug string) (int, error) {
	var count int

	query := `
		SELECT COUNT(*)
		FROM clicks
		WHERE slug = $1
	`

	err := postgresDB.QueryRow(
		context.Background(),
		query,
		slug,
	).Scan(&count)

	return count, err
}

func SaveLinkWithExpiry(slug string, originalURL string, expiresAt *time.Time) error {
	query := `
		INSERT INTO links (slug, original_url, expires_at)
		VALUES ($1, $2, $3)
	`

	_, err := postgresDB.Exec(
		context.Background(),
		query,
		slug,
		originalURL,
		expiresAt,
	)

	return err
}

func GetLinks() ([]models.Link, error) {
	query := `
		SELECT id, slug, original_url, created_at, expires_at
		FROM links
		ORDER BY created_at DESC
	`

	rows, err := postgresDB.Query(
		context.Background(),
		query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link

	for rows.Next() {
		var link models.Link

		err := rows.Scan(
			&link.ID,
			&link.Slug,
			&link.OriginalURL,
			&link.CreatedAt,
			&link.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}

		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

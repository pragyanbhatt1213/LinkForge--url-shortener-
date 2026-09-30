package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func CreateTables(db *pgx.Conn) error {
	query := `CREATE TABLE IF NOT EXISTS links(
			id BIGSERIAL PRIMARY KEY, 
			slug VARCHAR(50) UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMP);`
	_, err := db.Exec(context.Background(), query)

	clicksQuery := `CREATE TABLE IF NOT EXISTS clicks (
				id BIGSERIAL PRIMARY KEY,
				slug VARCHAR(50) NOT NULL,
				clicked_at TIMESTAMP NOT NULL DEFAULT NOW(),
				FOREIGN KEY (slug) REFERENCES links(slug));
`
	_, err = db.Exec(context.Background(), clicksQuery)
	return err

}

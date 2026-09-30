package config

import (
	"os"

	"github.com/jackc/pgx/v5"
)

func NewPostgresConfig() *pgx.ConnConfig {
	connString := "postgres://" +
		os.Getenv("POSTGRES_USER") + ":" +
		os.Getenv("POSTGRES_PASSWORD") + "@" +
		os.Getenv("POSTGRES_HOST") + ":" +
		os.Getenv("POSTGRES_PORT") + "/" +
		os.Getenv("POSTGRES_DB")

	config, err := pgx.ParseConfig(connString)
	if err != nil {
		panic(err)
	}

	return config
}

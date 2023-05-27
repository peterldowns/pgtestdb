package testdb

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx ddriver necessary for connecting
)

// Config contains the details needed to connect to a postgres server/database.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Options  string
}

func (c Config) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Options,
	)
}

func (c Config) connect() (*sql.DB, error) {
	db, err := sql.Open("pgx", c.URL())
	if err != nil {
		return nil, err
	}
	return db, nil
}

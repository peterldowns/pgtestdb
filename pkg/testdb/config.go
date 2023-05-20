package testdb

import (
	"database/sql"
	"fmt"
)

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

func (c Config) Connect() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.URL())
	if err != nil {
		return nil, err
	}
	return db, nil
}

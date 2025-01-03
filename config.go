package pgtestdb

import (
	"fmt"
	"net/url"
)

// ConfigFromURL is a helper function to create a Config from a connection string
// like "postgres://bob:secret@1.2.3.4:5432/mydb?sslmode=verify-full".
func ConfigFromURL(driverName, connString string, ops ...Option) (Config, error) {
	cfg, err := parseURL(connString)
	if err != nil {
		return Config{}, err
	}

	cfg.DriverName = driverName

	for _, op := range ops {
		op(&cfg)
	}

	return cfg, nil
}

func parseURL(connString string) (Config, error) {
	u, err := url.Parse(connString)
	if err != nil {
		return Config{}, err
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return Config{}, fmt.Errorf("invalid connection protocol: %s", u.Scheme)
	}

	cfg := Config{
		Host:    u.Hostname(),
		Port:    u.Port(),
		Options: u.RawQuery,
	}

	if len(u.Path) > 1 {
		cfg.Database = u.Path[1:]
	}

	if u.User != nil {
		cfg.User = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}

	return cfg, nil
}

// Option provides a way to configure the [Config] used by [ConfigFromURL].
type Option func(*Config)

// WithTestRole sets the role used to create and connect to the template database
// and each test database. See more [Config.TestRole].
func WithTestRole(role Role) Option {
	return func(cfg *Config) {
		cfg.TestRole = &role
	}
}

// WithForceTerminateConnections will force-disconnect any remaining
// database connections prior to dropping the test database. See more [Config.ForceTerminateConnections].
func WithForceTerminateConnections() Option {
	return func(cfg *Config) {
		cfg.ForceTerminateConnections = true
	}
}

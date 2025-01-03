package pgtestdb_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/peterldowns/pgtestdb"
)

func TestConfigFromURL(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		url := "postgres://bob:secret@1.2.3.4:5432/mydb?sslmode=verify-full"
		want := pgtestdb.Config{
			DriverName: "pgx",
			Host:       "1.2.3.4",
			Port:       "5432",
			User:       "bob",
			Password:   "secret",
			Database:   "mydb",
			Options:    "sslmode=verify-full",
		}

		cfg, err := pgtestdb.ConfigFromURL("pgx", url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if diff := cmp.Diff(want, cfg); diff != "" {
			t.Errorf("config mismatch (-want +got):\n%s", diff)
		}

		if cfg.URL() != url {
			t.Fatalf("unexpected URL, want %s, got %s", url, cfg.URL())
		}
	})

	t.Run("with options", func(t *testing.T) {
		url := "postgres://bob:secret@1.2.3.4:5432/mydb?sslmode=verify-full"
		testRole := pgtestdb.Role{
			Username:     "test",
			Password:     "test",
			Capabilities: "test",
		}
		want := pgtestdb.Config{
			DriverName:                "pgx",
			Host:                      "1.2.3.4",
			Port:                      "5432",
			User:                      "bob",
			Password:                  "secret",
			Database:                  "mydb",
			Options:                   "sslmode=verify-full",
			TestRole:                  &testRole,
			ForceTerminateConnections: true,
		}

		cfg, err := pgtestdb.ConfigFromURL("pgx", url, pgtestdb.WithForceTerminateConnections(), pgtestdb.WithTestRole(testRole))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if diff := cmp.Diff(want, cfg); diff != "" {
			t.Errorf("config mismatch (-want +got):\n%s", diff)
		}

		if cfg.URL() != url {
			t.Fatalf("unexpected URL, want %s, got %s", url, cfg.URL())
		}
	})

	t.Run("minimal", func(t *testing.T) {
		url := "postgres://localhost:5432"
		want := pgtestdb.Config{
			DriverName: "pgx",
			Host:       "localhost",
			Port:       "5432",
		}
		cfg, err := pgtestdb.ConfigFromURL("pgx", url)
		if err != nil {
			t.Fatalf("expected error: %s", err)
		}

		if diff := cmp.Diff(want, cfg); diff != "" {
			t.Errorf("config mismatch (-want +got):\n%s", diff)
		}

		if cfg.URL() != url {
			t.Fatalf("unexpected URL, want %s, got %s", url, cfg.URL())
		}
	})

	t.Run("bad protocol", func(t *testing.T) {
		url := "http://example.com"
		_, err := pgtestdb.ConfigFromURL("pgx", url)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

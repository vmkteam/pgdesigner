// Package pgre provides reverse engineering from a live PostgreSQL database.
package pgre

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// Options controls what objects to introspect.
type Options struct {
	Schemas []string // filter schemas (empty = all non-system)
	Full    bool     // include views, functions, triggers, extensions, domains, enums
}

// defaultSchemaFilter excludes system schemas from introspection queries.
const defaultSchemaFilter = "AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast') AND n.nspname NOT LIKE 'pg_temp_%'"

// Connect introspects a PostgreSQL database and returns a pgd.Project.
func Connect(dsn string, opts Options) (*pgd.Project, error) {
	db, err := connectDB(dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	intr := &introspector{db: db, ctx: context.Background(), opts: opts}
	return intr.introspect()
}

// connectDB parses DSN, connects, and verifies connectivity.
func connectDB(dsn string) (*pg.DB, error) {
	pgOpts, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}

	db := pg.Connect(pgOpts)

	if _, err := db.ExecOne("SELECT 1"); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, nil
}

// IsDSN returns true if the input looks like a PostgreSQL DSN.
func IsDSN(s string) bool {
	return strings.HasPrefix(s, "postgres://") || strings.HasPrefix(s, "postgresql://")
}

// DatabaseName returns the database name from a PostgreSQL DSN, or "" if it has none.
func DatabaseName(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Path, "/")
}

// RedactDSN hides the password of a PostgreSQL DSN for printing; other strings are returned as is.
func RedactDSN(s string) string {
	if !IsDSN(s) {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "postgres://(invalid DSN)"
	}
	return u.Redacted()
}

func parseDSN(dsn string) (*pg.Options, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // url.Error quotes the whole DSN with the password
		}
		return nil, err
	}

	opts := &pg.Options{
		Addr: u.Host,
	}

	if u.Port() == "" {
		opts.Addr = u.Hostname() + ":5432"
	}

	if u.User != nil {
		opts.User = u.User.Username()
		if p, ok := u.User.Password(); ok {
			opts.Password = p
		}
	}

	if len(u.Path) > 1 {
		opts.Database = u.Path[1:]
	}

	q := u.Query()
	if q.Get("sslmode") == "disable" {
		opts.TLSConfig = nil
	}

	return opts, nil
}

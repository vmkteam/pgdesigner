// Package pgre provides reverse engineering from a live PostgreSQL database.
package pgre

import (
	"context"
	"fmt"
	"net/url"

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
	return len(s) > 11 && (s[:11] == "postgres://" || s[:13] == "postgresql://")
}

func parseDSN(dsn string) (*pg.Options, error) {
	u, err := url.Parse(dsn)
	if err != nil {
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

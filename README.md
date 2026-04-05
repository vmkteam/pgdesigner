# PgDesigner

[![License: PolyForm Noncommercial](https://img.shields.io/badge/license-PolyForm%20NC-blue)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/vmkteam/pgdesigner)](https://github.com/vmkteam/pgdesigner/releases)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8)](go.mod)
[![macOS | Linux | Windows](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)]()

Visual PostgreSQL schema designer with git-friendly `.pgd` XML format, diff/ALTER engine, and 66 lint rules.

<p align="center">
  <a href="https://pgdesigner.io">
    <img src="https://pgdesigner.io/images/erd-dark.png" alt="PgDesigner ERD Canvas" width="800">
  </a>
</p>

<p align="center">
  <a href="https://pgdesigner.io">Website</a> &middot;
  <a href="https://demo.pgdesigner.io">Live Demo</a> &middot;
  <a href="https://pgdesigner.io/docs/quickstart">Docs</a> &middot;
  <a href="https://pgdesigner.io/docs/changelog">Changelog</a>
</p>

## Why PgDesigner

Most schema design tools are either database-agnostic (losing PG-specific features) or store schemas in binary formats that break git workflows. PgDesigner is built exclusively for PostgreSQL and solves both problems:

- **PG-specialized** — full PostgreSQL 18 DDL: partitions, RLS, domains, triggers, GIN/GiST indexes, identity columns
- **Git-friendly format** — `.pgd` XML that diffs cleanly in pull requests, no binary blobs
- **Diff/ALTER engine** — compare two schema versions, generate migration SQL with hazard detection
- **66 lint rules** — naming conventions, missing indexes, FK integrity, type checks — with autofix
- **No cloud, no account** — runs locally, your schemas stay on your machine

## Features

| Feature | Description |
|---------|-------------|
| **ERD Canvas** | Interactive schema diagram for 120+ tables, auto-layout, dark/light theme |
| **Table Editor** | Columns, constraints, indexes, FK with inline editing |
| **DDL Generation** | Complete CREATE/ALTER SQL from schema model |
| **Diff Engine** | Semantic ALTER between two schemas with hazard warnings |
| **Lint & Autofix** | 66 rules: naming, types, FK, indexes, constraints |
| **Sample Data** | Generate realistic INSERT statements from schema |
| **Import** | MicroOLAP PDD, DbSchema DBS, Toad DM2, plain SQL, live PostgreSQL |
| **Reverse Engineering** | Import from live PostgreSQL via `pg_catalog` |
| **CLI** | `generate`, `lint`, `diff`, `convert`, `merge` for CI/CD pipelines |

## Install

### Homebrew (macOS / Linux)

```bash
brew tap vmkteam/tap
brew install pgdesigner
```

### Docker

```bash
docker run --rm -p 9990:9990 -v "$PWD":/data ghcr.io/vmkteam/pgdesigner /data/schema.pgd
```

### Download

Pre-built binaries for macOS (arm64, amd64), Linux, and Windows are available on the [Releases](https://github.com/vmkteam/pgdesigner/releases) page.

## Quick Start

```bash
# Open schema in browser
pgdesigner schema.pgd

# Reverse-engineer from PostgreSQL
pgdesigner "postgres://user@localhost:5432/mydb?sslmode=disable"

# Generate DDL
pgdesigner generate schema.pgd > schema.sql

# Lint
pgdesigner lint schema.pgd

# Diff two schemas
pgdesigner diff old.pgd new.pgd

# Convert from other formats
pgdesigner convert schema.pdd -o schema.pgd
```

## Demo

Try PgDesigner without installing — [demo.pgdesigner.io](https://demo.pgdesigner.io) runs a read-only instance with the Chinook sample database.

## PGD Format

Git-friendly XML for PostgreSQL schemas. Covers tables, columns, indexes, FK, constraints, views, functions, triggers, sequences, enums, domains, composites, ranges, partitions, policies, roles, grants, comments, and diagram layouts.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<pgd version="1" pg-version="18" default-schema="public">
  <schema name="public">
    <table name="users">
      <column name="id" type="bigint" nullable="false">
        <identity generated="always"></identity>
      </column>
      <column name="email" type="varchar" length="255" nullable="false"></column>
      <pk name="pk_users">
        <column name="id"></column>
      </pk>
    </table>
  </schema>
</pgd>
```

Full spec: [docs/pgd-format/spec.md](docs/pgd-format/spec.md) | Coverage matrix: [docs/pgd-spec-coverage.md](docs/pgd-spec-coverage.md)

## Architecture

- **Backend:** Go — zenrpc JSON-RPC over HTTP
- **Frontend:** Vue 3.5 + Reka UI + Tailwind CSS + Vue Flow
- **Format:** `.pgd` XML — git-friendly, no binary blobs
- **No CGO** — SQL parsing via WebAssembly (wasilibs/go-pgquery)

## Development

```bash
make dev-backend     # Go server on :9990
make dev-frontend    # Vite on :5173
make test            # all tests
make build-full      # pnpm build + go build
make generate        # zenrpc codegen
make ts-client       # rpcgen -> TypeScript client
```

## Test Databases

Round-trip tested on 6 databases (SQL -> PGD -> DDL -> PostgreSQL -> pg_dump -> PGD -> diff = zero):

| Database | Tables | FK | Source |
|----------|-------:|---:|--------|
| Chinook | 11 | 11 | [lerocha/chinook-database](https://github.com/lerocha/chinook-database) |
| Northwind | 14 | 13 | [pthom/northwind_psql](https://github.com/pthom/northwind_psql) |
| Pagila | 15 | 18 | [devrimgunduz/pagila](https://github.com/devrimgunduz/pagila) |
| Airlines | 8 | 8 | [Postgres Pro Demo](https://postgrespro.com/community/demodb) |
| AdventureWorks | 68 | 89 | [lorint/AdventureWorks-for-Postgres](https://github.com/lorint/AdventureWorks-for-Postgres) |
| Synthetic | 8 | 9 | Custom (domains, GIN/GiST, triggers, RLS) |

## License

[PolyForm Noncommercial License 1.0.0](LICENSE) — free for non-commercial use.

For commercial use, see [pricing](https://pgdesigner.io/pricing) or [LICENSE-COMMERCIAL.md](LICENSE-COMMERCIAL.md).

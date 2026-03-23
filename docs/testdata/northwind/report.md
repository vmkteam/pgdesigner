# Northwind — Round-Trip Report

## Stats
- Tables: 14
- FK: 13
- PK: 13

## Pipeline Results

| Step | Status | Details |
|------|:------:|---------|
| 1. SQL → PGD | OK | All tables, PK, FK parsed |
| 2. PGD → SQL | OK | 226 lines generated |
| 3. SQL → Real DB | OK | No errors |
| 4. pg_dump | OK | 458 lines |
| 5. pg_dump → PGD | OK | roundtrip.pgd created |
| 6. diff | PARTIAL | FK name mismatch (semantic equivalent) |

## Issues Found & Fixed

### FIXED: FK without referenced column
Northwind uses `REFERENCES table` without `(column)`. Parser now falls back to FK column name,
then `resolveFKImplicitPK()` post-pass resolves to actual PK columns of target table.

### REMAINING: FK name mismatch in diff
Original SQL has named FKs (`fk_orders_customers`), pg_dump preserves them.
Diff shows DROP + ADD for all 13 FKs because constraint names differ between
schema.pgd (from original.sql) and roundtrip.pgd (from pg_dump).
**Semantically equivalent** — same columns, same actions, same tables.

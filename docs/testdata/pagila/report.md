# Pagila — Round-Trip Report

## Stats
- Tables: 15 + 7 partitions (22 in DDL)
- Indexes: 34
- FK: 18
- Views: 7
- Functions: 9 + 1 aggregate
- Triggers: 15
- Enums: 1 (mpaa_rating)
- Domains: 2

## Pipeline Results

| Step | Status | Details |
|------|:------:|---------|
| 1. SQL → PGD | OK | All objects including SETOF, AGGREGATE |
| 2. PGD → SQL | OK | 669 lines generated |
| 3. SQL → Real DB | OK | 0 errors |
| 4. pg_dump | OK | 1666 lines |
| 5. pg_dump → PGD | OK | roundtrip.pgd created |
| 6. diff | PARTIAL | Partition index name mismatch only |

## Issues Found & Fixed

### FIXED: function dependency order
Functions now generated before views (Phase 8 → Phase 9).
`sortFunctionsByDeps()` — topological sort by body references.

### FIXED: RETURNS SETOF
SQL parser now reads `TypeName.Setof` flag from pg_query AST.

### FIXED: CREATE AGGREGATE
Added `DefineStmt` handler in SQL converter for `OBJECT_AGGREGATE`.
Model stores aggregates as `Function` with `Kind="aggregate"` and
sfunc/stype/finalfunc/initcond fields. DDL generates `CREATE AGGREGATE`.

## Remaining Diff

Partition indexes have different names after pg_dump round-trip:
- Original: `idx_fk_payment_p2022_01_customer_id`
- pg_dump: `payment_p2022_01_customer_id_idx`

This is cosmetic — semantically equivalent.

# AdventureWorks — Round-Trip Report

## Stats
- Tables: 68 (5 schemas: humanresources, person, production, purchasing, sales)
- FK: 89
- Indexes: 2
- Domains: 6 (AccountNumber, Flag, Name, NameStyle, OrderNumber, Phone)
- Views: 20
- Comments: 100+
- Functions: 10+

## Pipeline Results

| Step | Status | Details |
|------|:------:|---------|
| 1. SQL → PGD | OK | All 68 tables, 89 FK, 5 schemas, 6 domains |
| 2. PGD → SQL | OK | 2176 lines generated |
| 3. SQL → Real DB | OK | 0 errors |
| 4. pg_dump | OK | schema-only |
| 5. pg_dump → PGD | OK | |
| 6. diff | OK | **no changes** — perfect round-trip |

## Issues Fixed During Testing

### FIXED: Mixed-case domain types not quoted
Domain types like `Flag`, `NameStyle` stored with uppercase in model.
DDL generator wrote `Flag` (unquoted) → PG lowercased to `flag` → not found.
Fixed: `quoteType()` now detects uppercase and quotes custom types.

## Notes
- Largest multi-schema test (5 schemas, 68 tables, 89 FK)
- Cross-schema FK references work correctly
- Domain types used across schemas (public.Flag in humanresources.employee)
- 20 views with complex cross-schema JOINs including xpath expressions

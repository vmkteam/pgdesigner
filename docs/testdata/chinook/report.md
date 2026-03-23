# Chinook — Round-Trip Report

## Stats
- Tables: 11
- FK: 11
- PK: 11
- Indexes: 11

## Pipeline Results

| Step | Status | Details |
|------|:------:|---------|
| 1. SQL → PGD | OK | All tables, PK, FK, indexes parsed |
| 2. PGD → SQL | OK | 218 lines generated |
| 3. SQL → Real DB | OK | No errors |
| 4. pg_dump | OK | 475 lines |
| 5. pg_dump → PGD | OK | roundtrip.pgd created |
| 6. diff | OK | **no changes** — perfect round-trip |

## Notes

Chinook is a clean, simple schema (music store) with no advanced features.
All 11 tables have PK, FK, and indexes. No views, functions, triggers, enums, or domains.
This is the first test database with a perfect round-trip (zero diff).

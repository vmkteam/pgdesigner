# Airlines (PostgresPro Demo) — Round-Trip Report

## Stats
- Tables: 8 (bookings schema)
- FK: 8
- Indexes: 0 (pg_dump schema-only drops them)
- Views: 3
- Functions: 1 (now)
- Comments: 30+

## Pipeline Results

| Step | Status | Details |
|------|:------:|---------|
| 1. SQL → PGD | OK | All tables, FK, views, function, comments |
| 2. PGD → SQL | OK | 253 lines generated |
| 3. SQL → Real DB | OK | No errors |
| 4. pg_dump | OK | 1121 lines |
| 5. pg_dump → PGD | OK | roundtrip.pgd created |
| 6. diff | OK | **no changes** — perfect round-trip |

## Issues Fixed During Testing

### FIXED: COMMENT ON SCHEMA / COMMENT ON FUNCTION
pg_query AST uses `String_` node for COMMENT ON SCHEMA and `ObjectWithArgs` node
for COMMENT ON FUNCTION — these were not handled by the comment parser.
Added fallback handling in `convertComment()`.

## Notes
- Single non-public schema (`bookings`) — good multi-schema comment test
- Heavy use of COMMENT ON (tables, columns, views, schema, function)
- Views with complex CTEs (routes view)

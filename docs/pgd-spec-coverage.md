# PGD Spec Coverage

How well each layer supports the [PGD format spec](pgd-format/spec.md) (22 sections).

| Spec Section | Read | Write | DDL Gen | SQL Parse | RE | UI |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| 1. Project metadata | + | + | — | + | — | + |
| 2. Database | + | + | — | — | — | — |
| 3. Roles | + | + | + | — | — | — |
| 4. Tablespaces | + | + | + | — | — | — |
| 5. Extensions | + | + | + | + | + | + |
| 6. Types (enum) | + | + | + | + | + | + |
| 6. Types (domain) | + | + | + | + | + | + |
| 6. Types (composite) | + | + | + | + | — | — |
| 6. Types (range) | + | + | + | — | — | — |
| 7. Sequences | + | + | + | + | + | + |
| 8. Schemas | + | + | + | + | + | + |
| 9. Tables | + | + | + | + | + | + |
| 10. Columns | + | + | + | + | + | + |
| 10. Identity | + | + | + | + | + | + |
| 10. Generated (stored) | + | + | + | + | + | + |
| 10. Collation | + | + | + | + | + | — |
| 10. Compression | + | + | + | — | + | — |
| 10. Storage | + | + | + | — | + | — |
| 11. PK | + | + | + | + | + | + |
| 11. FK | + | + | + | + | + | + |
| 11. Unique | + | + | + | + | + | + |
| 11. Check | + | + | + | + | + | + |
| 11. Exclude | + | + | + | + | + | + |
| 12. Storage params (WITH) | + | + | + | — | — | — |
| 13. Partitioning | + | + | + | + | + | + |
| 14. Indexes | + | + | + | + | + | + |
| 14. Expression indexes | + | + | + | + | + | — |
| 14. Partial indexes (WHERE) | + | + | + | + | + | — |
| 14. INCLUDE | + | + | + | + | + | — |
| 15. Views | + | + | + | + | + | + |
| 15. Materialized views | + | + | + | + | + | + |
| 16. Functions | + | + | + | + | + | + |
| 16. Aggregates | + | + | + | + | — | — |
| 17. Triggers | + | + | + | + | + | + |
| 18. Policies (RLS) | + | + | + | — | — | — |
| 19. Comments | + | + | + | + | + | + |
| 20. Grants | + | + | + | — | — | — |
| 21. Rules (deprecated) | + | + | — | — | — | — |
| 22. Layouts | + | + | — | — | — | + |

**Legend:** Read = XML unmarshal, Write = XML marshal, DDL Gen = SQL output, SQL Parse = pg_dump/SQL import, RE = reverse engineering from live PG, UI = visual editor.

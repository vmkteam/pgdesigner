# PGD Format Specification

**Version:** 1
**Target:** PostgreSQL 18
**File Extension:** `.pgd`
**Encoding:** UTF-8
**Validation:** `pgd-format.xsd` (see `lfdfq pgd-format.xsd`)

## Overview

PGD is a git-friendly XML format for describing PostgreSQL database schemas. It provides structural modeling for all major PostgreSQL DDL constructs (tables, indexes, types, sequences, views, functions, triggers, policies, roles, extensions, grants, comments) and includes diagram layout metadata for visual schema design.

### Design Principles

- **Git-friendly** — single-line attributes, predictable element ordering, clean diffs
- **Complete PG18 coverage** — all DDL constructs modeled structurally
- **Raw SQL only where necessary** — function bodies, CHECK expressions, WHERE predicates, partition bounds
- **Layout separated from model** — diagram metadata in `<layouts>`, can be split to a separate file
- **No IDs** — objects referenced by name (like SQL itself), no artificial numeric identifiers

## Document Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<pgd version="1" pg-version="18" default-schema="public">
  <project>...</project>
  <database>...</database>
  <roles>...</roles>
  <tablespaces>...</tablespaces>
  <extensions>...</extensions>
  <types>...</types>
  <sequences>...</sequences>
  <schema name="public">...</schema>
  <schema name="app">...</schema>
  <views>...</views>
  <functions>...</functions>
  <triggers>...</triggers>
  <rules>...</rules>
  <policies>...</policies>
  <comments>...</comments>
  <grants>...</grants>
  <layouts>...</layouts>
</pgd>
```

All top-level children are optional. The order shown above is the canonical order.

## Root Element: `<pgd>`

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `version` | int | yes | — | Format version (currently `1`) |
| `pg-version` | string | no | — | Target PostgreSQL major version (`14`–`18`) |
| `default-schema` | string | no | `public` | Default schema when schema is omitted |

---

## 1. Project Metadata: `<project>`

```xml
<project name="myproject" description="My database schema">
  <settings>
    <naming convention="snake_case" tables="plural"></naming>
    <defaults nullable="true" on-delete="restrict" on-update="restrict"></defaults>
    <lint ignore-rules="W015,I009"></lint>
  </settings>
</project>
```

### `<project>` attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Project name |
| `description` | string | no | Project description |

### `<naming>` attributes

| Attribute | Type | Default | Values |
|-----------|------|---------|--------|
| `convention` | enum | `camelCase` | `camelCase`, `snake_case`, `PascalCase` |
| `tables` | enum | — | `plural`, `singular` (empty = no check) |

### `<defaults>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `nullable` | boolean | `true` | Default column nullability |
| `on-delete` | enum | `restrict` | Default FK ON DELETE action |
| `on-update` | enum | `restrict` | Default FK ON UPDATE action |

### `<lint>` attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `ignore-rules` | string | Comma-separated lint rule codes to suppress project-wide |

---

## 2. Database: `<database>`

```xml
<database name="mydb" encoding="UTF8" collation="en_US.UTF-8"
          ctype="en_US.UTF-8" template="template0" owner="admin"></database>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Database name |
| `encoding` | string | no | Character encoding (e.g. `UTF8`) |
| `collation` | string | no | LC_COLLATE setting |
| `ctype` | string | no | LC_CTYPE setting |
| `icu-locale` | string | no | ICU collation (PG15+) |
| `locale` | string | no | Locale shorthand |
| `template` | string | no | Template database |
| `tablespace` | string | no | Default tablespace |
| `owner` | string | no | Database owner role |

---

## 3. Roles: `<roles>`

```xml
<roles>
  <role name="app_user" login="true" inherit="true">
    <in-role name="app_readers"></in-role>
  </role>
  <role name="app_readers" login="false"></role>
</roles>
```

### `<role>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Role name |
| `login` | boolean | `false` | Can log in |
| `inherit` | boolean | `true` | Inherits privileges of granted roles |
| `createdb` | boolean | `false` | Can create databases |
| `createrole` | boolean | `false` | Can create roles |
| `superuser` | boolean | `false` | Superuser privileges |
| `replication` | boolean | `false` | Can initiate streaming replication |
| `bypassrls` | boolean | `false` | Bypasses row-level security |
| `connection-limit` | int | `-1` | Max connections (`-1` = unlimited) |
| `password-encrypted` | boolean | `true` | Password stored encrypted |
| `valid-until` | string | — | Expiration timestamp (ISO 8601) |

### `<in-role>` child element

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Parent role to grant membership in |

---

## 4. Tablespaces: `<tablespaces>`

```xml
<tablespaces>
  <tablespace name="fast_ssd" location="/mnt/ssd/pg_data" owner="postgres"></tablespace>
</tablespaces>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Tablespace name |
| `location` | string | yes | Filesystem path |
| `owner` | string | no | Owner role |

---

## 5. Extensions: `<extensions>`

```xml
<extensions>
  <extension name="uuid-ossp" schema="public"></extension>
  <extension name="pgcrypto"></extension>
</extensions>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Extension name |
| `schema` | string | no | Install into schema |
| `version` | string | no | Specific version |
| `cascade` | boolean | no | Auto-install dependencies |

---

## 6. Types: `<types>`

Contains four sub-sections: `<enum>`, `<composite>`, `<domain>`, `<range>`.

### 6.1 Enum

```xml
<enum name="status" schema="public">
  <label>active</label>
  <label>inactive</label>
  <label>deleted</label>
</enum>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Type name |
| `schema` | string | no | Schema |

Child `<label>` elements contain enum value strings in order.

### 6.2 Composite

```xml
<composite name="address" schema="public">
  <field name="street" type="text"></field>
  <field name="city" type="varchar" length="100"></field>
  <field name="zip" type="varchar" length="10"></field>
</composite>
```

**`<field>` attributes:**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Field name |
| `type` | string | yes | Data type |
| `length` | int | no | Type length (for `varchar`, `char`) |
| `collation` | string | no | Collation |

### 6.3 Domain

```xml
<domain name="email" schema="public" type="varchar" length="255" default="''">
  <not-null></not-null>
  <constraint name="email_check"><![CDATA[VALUE ~ '^.+@.+$']]></constraint>
</domain>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Domain name |
| `schema` | string | no | Schema |
| `type` | string | yes | Base type |
| `length` | int | no | Type length |
| `precision` | int | no | Numeric precision |
| `scale` | int | no | Numeric scale |
| `collation` | string | no | Collation |
| `default` | string | no | Default expression (raw SQL) |

Child `<not-null/>` — presence means NOT NULL.
Child `<constraint>` — named CHECK constraint; text content is the expression (CDATA).

### 6.4 Range

```xml
<range name="float_range" subtype="float8"></range>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Type name |
| `schema` | string | no | Schema |
| `subtype` | string | yes | Underlying element type |
| `subtype-opclass` | string | no | Operator class |
| `collation` | string | no | Collation |
| `canonical` | string | no | Canonical function |
| `subtype-diff` | string | no | Difference function |
| `multirange-type-name` | string | no | Multirange type name (PG14+) |

---

## 7. Sequences: `<sequences>`

```xml
<sequences>
  <sequence name="order_seq" schema="public" as="bigint"
            start="1000" increment="1" cache="10"></sequence>
</sequences>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Sequence name |
| `schema` | string | — | Schema |
| `as` | string | — | Type: `smallint`, `integer`, `bigint` |
| `start` | long | `1` | Start value |
| `increment` | long | `1` | Increment |
| `min` | long | — | Minimum value |
| `max` | long | — | Maximum value |
| `cache` | long | `1` | Cache size |
| `cycle` | boolean | `false` | Wrap around on limit |
| `owned-by` | string | — | Owner column (`table.column`) |

---

## 8. Schemas: `<schema>`

A schema contains tables and indexes. Multiple `<schema>` elements can appear at the top level.

```xml
<schema name="public">
  <table name="users">...</table>
  <table name="orders">...</table>
  <index name="idx_orders_user_id" table="orders">...</index>
</schema>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Schema name |

**Note:** Indexes are stored at the schema level, not inside tables.

---

## 9. Tables: `<table>`

```xml
<table name="users" comment="Application users">
  <column name="id" type="bigint" nullable="false">
    <identity generated="always"></identity>
  </column>
  <column name="email" type="varchar" length="255" nullable="false"></column>
  <column name="name" type="text"></column>
  <column name="created_at" type="timestamptz" nullable="false" default="now()"></column>
  <pk name="pk_users">
    <column name="id"></column>
  </pk>
  <unique name="uq_users_email">
    <column name="email"></column>
  </unique>
  <check name="chk_email"><![CDATA[email <> '']]></check>
</table>
```

### `<table>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Table name |
| `unlogged` | boolean | `false` | UNLOGGED table |
| `temporary` | boolean | `false` | TEMPORARY table |
| `on-commit` | string | — | `preserve-rows`, `delete-rows`, `drop` |
| `tablespace` | string | — | Tablespace name |
| `comment` | string | — | Table comment (shorthand for COMMENT ON TABLE) |
| `row-level-security` | boolean | `false` | Enable RLS |
| `force-row-level-security` | boolean | `false` | Force RLS for table owner |
| `partition-of` | string | — | Parent table name (Variant A, deprecated) |
| `inherits` | string | — | Comma-separated parent table names |
| `using` | string | — | Table access method (e.g. `heap`) |
| `generate` | string | — | PGDesigner: `false` to skip DDL generation |
| `lint-ignore` | string | — | PGDesigner: comma-separated lint rule codes |

### Child elements (in order)

1. `<column>` — column definitions (must come first)
2. `<pk>` — primary key (at most one)
3. `<fk>` — foreign keys
4. `<unique>` — unique constraints
5. `<check>` — check constraints
6. `<exclude>` — exclusion constraints
7. `<with>` — storage parameters
8. `<partition-by>` — partition key definition
9. `<partition-bound>` — partition bound (Variant A, deprecated)
10. `<partition>` — child partitions (Variant B)

---

## 10. Columns: `<column>`

```xml
<column name="price" type="numeric" precision="10" scale="2"
        nullable="false" default="0" comment="Product price in cents"></column>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Column name |
| `type` | string | (required) | Data type (append `[]` for arrays: `integer[]`) |
| `length` | int | — | Length for `varchar(n)`, `char(n)`, `bit(n)` |
| `precision` | int | — | Total digits for `numeric(p,s)` |
| `scale` | int | — | Decimal digits for `numeric(p,s)` |
| `nullable` | boolean | `true` | Allows NULL (`false` = NOT NULL) |
| `default` | string | — | Default expression (raw SQL) |
| `collation` | string | — | COLLATE collation_name |
| `storage` | enum | — | `plain`, `external`, `extended`, `main`, `default` |
| `compression` | enum | — | `lz4`, `pglz`, `default` (PG14+) |
| `comment` | string | — | Column comment (shorthand for COMMENT ON COLUMN) |

### Supported Data Types

All PostgreSQL built-in types are supported as string values:

| Category | Types |
|----------|-------|
| Integer | `smallint`, `integer`, `bigint` |
| Serial | `smallserial`, `serial`, `bigserial` |
| Decimal | `numeric`, `decimal`, `real`, `double precision`, `money` |
| Character | `varchar`, `character varying`, `char`, `character`, `text` |
| Binary | `bytea` |
| Boolean | `boolean` |
| Date/Time | `date`, `time`, `timetz`, `timestamp`, `timestamptz`, `interval` |
| UUID | `uuid` |
| JSON | `json`, `jsonb` |
| XML | `xml` |
| Network | `inet`, `cidr`, `macaddr`, `macaddr8` |
| Geometric | `point`, `line`, `lseg`, `box`, `path`, `polygon`, `circle` |
| Text Search | `tsvector`, `tsquery` |
| Bit String | `bit`, `varbit`, `bit varying` |
| Range | `int4range`, `int8range`, `numrange`, `tsrange`, `tstzrange`, `daterange` |
| Multirange | `int4multirange`, `int8multirange`, `nummultirange`, `tsmultirange`, `tstzmultirange`, `datemultirange` |
| Other | `pg_lsn`, `pg_snapshot`, `oid`, `regclass`, `regtype` |

Custom types (enums, composites, domains) are referenced by name. Schema-qualified types use dot notation: `myschema.my_type`.

### 10.1 Identity Column: `<identity>`

```xml
<column name="id" type="bigint" nullable="false">
  <identity generated="always">
    <sequence start="100" increment="1" cache="10"></sequence>
  </identity>
</column>
```

**`<identity>` attributes:**

| Attribute | Type | Required | Values |
|-----------|------|----------|--------|
| `generated` | enum | yes | `always`, `by-default` |

**`<sequence>` child (optional):**

| Attribute | Type | Description |
|-----------|------|-------------|
| `start` | long | Start value |
| `increment` | long | Increment |
| `min` | long | Minimum value |
| `max` | long | Maximum value |
| `cache` | long | Cache size |
| `cycle` | boolean | Wrap around |

### 10.2 Generated Column: `<generated>`

```xml
<column name="full_name" type="text">
  <generated expression="first_name || ' ' || last_name" stored="true"></generated>
</column>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `expression` | string | (required) | SQL expression |
| `stored` | boolean | `true` | `true` = STORED, `false` = VIRTUAL (PG18+) |

---

## 11. Constraints

### 11.1 Primary Key: `<pk>`

```xml
<pk name="pk_users" without-overlaps="false">
  <column name="id"></column>
</pk>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | — | Constraint name |
| `without-overlaps` | boolean | `false` | WITHOUT OVERLAPS (PG17+, temporal) |
| `deferrable` | boolean | `false` | DEFERRABLE |
| `initially` | enum | — | `immediate`, `deferred` |
| `enforced` | boolean | `true` | NOT ENFORCED (PG18+) |

### 11.2 Foreign Key: `<fk>`

```xml
<fk name="fk_orders_user" to-table="users" on-delete="cascade" on-update="no action">
  <column name="user_id" references="id"></column>
</fk>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | — | Constraint name |
| `to-table` | string | (required) | Referenced table (can be `schema.table`) |
| `to-schema` | string | — | Referenced table schema (if different) |
| `on-delete` | enum | `restrict` | `restrict`, `cascade`, `set-null`, `set-default`, `no action` |
| `on-update` | enum | `restrict` | Same as on-delete |
| `deferrable` | boolean | `false` | DEFERRABLE |
| `initially` | enum | — | `immediate`, `deferred` |
| `match` | enum | `simple` | `simple`, `full`, `partial` |
| `enforced` | boolean | `true` | NOT ENFORCED (PG18+) |
| `period` | string | — | Column name for temporal FK (PG17+) |

**`<column>` child attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Local column name |
| `references` | string | Referenced column name |

### 11.3 Unique: `<unique>`

```xml
<unique name="uq_users_email" nulls-distinct="false">
  <column name="email"></column>
</unique>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | — | Constraint name |
| `nulls-distinct` | boolean | `true` | `false` = NULLS NOT DISTINCT (PG15+) |
| `without-overlaps` | boolean | `false` | WITHOUT OVERLAPS (PG17+) |
| `deferrable` | boolean | `false` | DEFERRABLE |
| `initially` | enum | — | `immediate`, `deferred` |
| `enforced` | boolean | `true` | NOT ENFORCED (PG18+) |

### 11.4 Check: `<check>`

```xml
<check name="chk_positive_amount" no-inherit="false"><![CDATA[amount > 0]]></check>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | — | Constraint name |
| `no-inherit` | boolean | `false` | NO INHERIT |
| `enforced` | boolean | `true` | NOT ENFORCED (PG18+) |

The element text content is the CHECK expression (use CDATA for special characters).

### 11.5 Exclude: `<exclude>`

```xml
<exclude name="excl_booking_overlap" using="gist">
  <element column="room_id" with="="></element>
  <element column="period" with="&amp;&amp;"></element>
</exclude>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | — | Constraint name |
| `using` | string | `gist` | Index method |
| `deferrable` | boolean | `false` | DEFERRABLE |
| `initially` | enum | — | `immediate`, `deferred` |
| `enforced` | boolean | `true` | NOT ENFORCED (PG18+) |

**`<element>` attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `column` | string | Column name |
| `expression` | string | Expression (alternative to column) |
| `with` | string | Operator (e.g. `=`, `&&`, `<>`) |

---

## 12. Storage Parameters: `<with>`

```xml
<with>
  <param name="fillfactor" value="90"></param>
  <param name="autovacuum_enabled" value="false"></param>
</with>
```

Used in `<table>`, `<partition>`, `<materialized-view>`, and `<index>`.

---

## 13. Partitioning

PGD supports two variants for partition children. **Variant B** (nested) is the current standard.

### 13.1 Partition Key: `<partition-by>`

```xml
<partition-by type="range">
  <column name="created_at"></column>
</partition-by>
```

| Attribute | Type | Required | Values |
|-----------|------|----------|--------|
| `type` | enum | yes | `range`, `list`, `hash` |

Children: `<column>` (by name) and/or `<expression>` (raw SQL).

### 13.2 Child Partitions: `<partition>` (Variant B)

```xml
<table name="events">
  <column name="id" type="bigint" nullable="false"></column>
  <column name="created_at" type="timestamptz" nullable="false"></column>
  <pk name="pk_events">
    <column name="id"></column>
    <column name="created_at"></column>
  </pk>
  <partition-by type="range">
    <column name="created_at"></column>
  </partition-by>
  <partition name="events_2024">
    <bound>FOR VALUES FROM ('2024-01-01') TO ('2025-01-01')</bound>
  </partition>
  <partition name="events_2025">
    <bound>FOR VALUES FROM ('2025-01-01') TO ('2026-01-01')</bound>
  </partition>
  <partition name="events_default">
    <bound>DEFAULT</bound>
  </partition>
</table>
```

**`<partition>` attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Child table name |
| `tablespace` | string | Tablespace override |

**`<partition>` children:**

| Element | Description |
|---------|-------------|
| `<bound>` | Partition bound clause (raw SQL: `FOR VALUES ...` or `DEFAULT`) |
| `<partition-by>` | Sub-partition key (for multi-level partitioning) |
| `<partition>` | Sub-partition children (recursive) |
| `<with>` | Storage parameters |

### 13.3 Multi-level Partitioning

```xml
<partition name="events_2024">
  <bound>FOR VALUES FROM ('2024-01-01') TO ('2025-01-01')</bound>
  <partition-by type="list">
    <column name="region"></column>
  </partition-by>
  <partition name="events_2024_us">
    <bound>FOR VALUES IN ('us')</bound>
  </partition>
  <partition name="events_2024_eu">
    <bound>FOR VALUES IN ('eu')</bound>
  </partition>
</partition>
```

### 13.4 Variant A (Deprecated)

Variant A uses `partition-of` attribute on separate top-level table elements. This is deprecated; use Variant B (nested `<partition>` elements) instead. `MigratePartitions()` converts Variant A to Variant B automatically.

---

## 14. Indexes: `<index>`

Indexes are stored at the **schema level**, not inside tables.

```xml
<index name="idx_orders_user_id" table="orders">
  <column name="user_id"></column>
</index>

<index name="idx_orders_email_lower" table="orders" unique="true" using="btree">
  <expression><![CDATA[lower(email)]]></expression>
  <where><![CDATA[deleted_at IS NULL]]></where>
</index>

<index name="idx_orders_covering" table="orders" unique="true">
  <column name="order_number"></column>
  <include>
    <column name="status"></column>
    <column name="total"></column>
  </include>
</index>
```

### `<index>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Index name |
| `table` | string | (required) | Table name |
| `schema` | string | — | Table schema (if different from parent `<schema>`) |
| `unique` | boolean | `false` | UNIQUE index |
| `using` | enum | `btree` | `btree`, `hash`, `gist`, `spgist`, `gin`, `brin` |
| `nulls-distinct` | boolean | `true` | `false` = NULLS NOT DISTINCT (PG15+) |
| `tablespace` | string | — | Tablespace |
| `concurrently` | boolean | `false` | CONCURRENTLY (used by ALTER generator) |

### Index `<column>` attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Column name |
| `order` | enum | `asc`, `desc` |
| `nulls` | enum | `first`, `last` |
| `opclass` | string | Operator class (e.g. `jsonb_path_ops`, `text_pattern_ops`) |

### `<expression>` — expression index key (CDATA)

### `<where>` — partial index predicate (CDATA)

### `<include>` — covering index columns (PG11+)

Contains `<column>` elements (only `name` attribute used).

---

## 15. Views: `<views>`

### 15.1 View: `<view>`

```xml
<view name="v_active_users" schema="public">
  <query><![CDATA[SELECT id, name FROM users WHERE deleted_at IS NULL]]></query>
</view>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | View name |
| `schema` | string | — | Schema |
| `recursive` | boolean | `false` | RECURSIVE view |
| `security-barrier` | boolean | `false` | security_barrier option |
| `security-invoker` | boolean | `false` | security_invoker option (PG15+) |
| `check-option` | string | — | `local`, `cascaded` |

### 15.2 Materialized View: `<materialized-view>`

```xml
<materialized-view name="mv_daily_stats" schema="public" tablespace="fast_ssd">
  <query><![CDATA[SELECT date, count(*) FROM events GROUP BY date]]></query>
</materialized-view>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | View name |
| `schema` | string | — | Schema |
| `tablespace` | string | — | Tablespace |
| `using` | string | — | Access method |
| `with-data` | boolean | `true` | `false` = WITH NO DATA |

---

## 16. Functions: `<functions>`

```xml
<function name="get_user_count" schema="public" returns="integer"
          language="sql" volatility="stable" parallel="safe">
  <body><![CDATA[SELECT count(*)::integer FROM users;]]></body>
</function>

<function name="create_user" schema="public" returns="bigint"
          language="plpgsql" volatility="volatile" security="definer">
  <arg name="p_name" type="text"></arg>
  <arg name="p_email" type="text"></arg>
  <arg name="p_role" type="text" default="'user'"></arg>
  <body><![CDATA[
DECLARE
  v_id bigint;
BEGIN
  INSERT INTO users (name, email, role) VALUES (p_name, p_email, p_role) RETURNING id INTO v_id;
  RETURN v_id;
END;
  ]]></body>
</function>
```

### `<function>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Function name |
| `schema` | string | — | Schema |
| `kind` | string | `function` | `function`, `procedure`, `aggregate` |
| `returns` | string | — | Return type (omit when using `<returns-table>`) |
| `language` | string | (required) | `plpgsql`, `sql`, `python`, etc. |
| `volatility` | enum | `volatile` | `immutable`, `stable`, `volatile` |
| `security` | enum | `invoker` | `invoker`, `definer` |
| `parallel` | enum | `unsafe` | `unsafe`, `restricted`, `safe` |
| `strict` | boolean | `false` | RETURNS NULL ON NULL INPUT |
| `leakproof` | boolean | `false` | LEAKPROOF |
| `window` | boolean | `false` | Window function |
| `cost` | int | — | Execution cost estimate |
| `rows` | int | — | Estimated rows returned |

### `<arg>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | — | Argument name |
| `type` | string | (required) | Data type |
| `mode` | string | `in` | `in`, `out`, `inout`, `variadic` |
| `default` | string | — | Default expression |

### `<returns-table>` (alternative to `returns` attribute)

```xml
<returns-table>
  <column name="id" type="bigint"></column>
  <column name="name" type="text"></column>
</returns-table>
```

### Aggregate functions (`kind="aggregate"`)

Additional attributes: `sfunc`, `stype`, `finalfunc`, `combinefunc`, `initcond`, `sortop`.

---

## 17. Triggers: `<triggers>`

```xml
<trigger name="trg_users_updated" schema="public" table="users"
         timing="before" events="update" for-each="row">
  <execute function="set_updated_at"></execute>
</trigger>

<trigger name="trg_audit_insert" schema="public" table="orders"
         timing="after" events="insert,update,delete" for-each="row">
  <referencing new-table="new_orders" old-table="old_orders"></referencing>
  <when><![CDATA[NEW.status <> OLD.status]]></when>
  <execute function="audit_changes"></execute>
</trigger>
```

### `<trigger>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Trigger name |
| `schema` | string | — | Schema |
| `table` | string | (required) | Table name |
| `timing` | enum | (required) | `before`, `after`, `instead-of` |
| `events` | string | (required) | Comma-separated: `insert`, `update`, `delete`, `truncate` |
| `for-each` | enum | `row` | `row`, `statement` |
| `constraint` | boolean | `false` | CONSTRAINT trigger |
| `deferrable` | boolean | `false` | DEFERRABLE (constraint triggers) |
| `initially` | enum | — | `immediate`, `deferred` |
| `of-columns` | string | — | UPDATE OF columns (comma-separated) |

### `<referencing>` (transition tables)

| Attribute | Type | Description |
|-----------|------|-------------|
| `old-table` | string | OLD TABLE alias |
| `new-table` | string | NEW TABLE alias |

### `<when>` — trigger condition (CDATA)

### `<execute>`

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `function` | string | yes | Function name to execute |
| `arguments` | string | no | Arguments string |

---

## 18. Policies: `<policies>` (Row-Level Security)

```xml
<policies>
  <policy name="users_own_data" schema="public" table="users"
          type="permissive" command="all" to="app_user">
    <using><![CDATA[id = current_user_id()]]></using>
    <with-check><![CDATA[id = current_user_id()]]></with-check>
  </policy>
</policies>
```

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Policy name |
| `schema` | string | — | Schema |
| `table` | string | (required) | Table name |
| `type` | enum | `permissive` | `permissive`, `restrictive` |
| `command` | enum | `all` | `all`, `select`, `insert`, `update`, `delete` |
| `to` | string | `public` | Comma-separated role names |

### `<using>` — USING expression (CDATA)
### `<with-check>` — WITH CHECK expression (CDATA)

---

## 19. Comments: `<comments>`

For comments on objects that don't have a `comment` attribute shorthand (or for long-form comments on any object).

```xml
<comments>
  <comment on="schema" name="app">Application schema</comment>
  <comment on="function" schema="public" name="get_user_count">Returns total user count</comment>
  <comment on="extension" name="pgcrypto">Cryptographic functions</comment>
</comments>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `on` | enum | yes | Object type (see below) |
| `schema` | string | no | Schema (where applicable) |
| `table` | string | no | Table (for `column`, `trigger`, `policy`, `constraint`) |
| `name` | string | yes | Object name |

**Supported `on` values:** `schema`, `table`, `column`, `index`, `sequence`, `view`, `materialized-view`, `function`, `trigger`, `policy`, `type`, `domain`, `extension`, `role`, `database`, `constraint`.

The element text content is the comment string.

**Note:** Tables and columns support a `comment` attribute shorthand directly on `<table>` and `<column>` elements. The `<comments>` section is for other object types or when you prefer centralized comments.

---

## 20. Grants: `<grants>`

```xml
<grants>
  <grant on="schema" name="app" privileges="usage" to="app_user"></grant>
  <grant on="all-tables-in-schema" schema="app" privileges="select,insert,update,delete" to="app_user"></grant>
  <grant on="table" schema="public" name="users" privileges="select" to="readonly"></grant>
  <grant-role role="app_readers" to="app_user" with-inherit="true"></grant-role>
</grants>
```

### `<grant>` attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `on` | enum | yes | Target type (see below) |
| `schema` | string | no | Schema |
| `name` | string | no | Object name (omitted for `all-*-in-schema`) |
| `privileges` | string | yes | Comma-separated: `select`, `insert`, `update`, `delete`, `truncate`, `references`, `trigger`, `maintain`, `usage`, `create`, `execute`, `connect`, `temporary`, `all` |
| `to` | string | yes | Role name or `public` |
| `with-grant-option` | boolean | no | WITH GRANT OPTION |

**Supported `on` values:** `schema`, `table`, `all-tables-in-schema`, `sequence`, `all-sequences-in-schema`, `function`, `all-functions-in-schema`, `type`, `database`.

### `<grant-role>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `role` | string | (required) | Role to grant |
| `to` | string | (required) | Grantee role |
| `with-admin` | boolean | `false` | WITH ADMIN OPTION |
| `with-inherit` | boolean | `true` | WITH INHERIT (PG16+) |
| `with-set` | boolean | `true` | WITH SET (PG16+) |

---

## 21. Rules: `<rules>` (Deprecated)

```xml
<rules>
  <rule name="rule_name" schema="public" table="users" event="insert" instead="true">
    <![CDATA[INSERT INTO audit_log (table_name) VALUES ('users')]]>
  </rule>
</rules>
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Rule name |
| `schema` | string | no | Schema |
| `table` | string | yes | Table name |
| `event` | string | yes | `select`, `insert`, `update`, `delete` |
| `instead` | string | no | `true` = INSTEAD rule |
| `where` | string | no | Condition |

**Note:** Rules are deprecated in PostgreSQL in favor of triggers. This element is retained for reverse-engineering compatibility.

---

## 22. Layouts: `<layouts>` (PGDesigner Metadata)

Diagram layout metadata is separated from the data model. It stores table positions, visual groups, and notes for the ERD canvas.

```xml
<layouts>
  <layout name="Default Diagram" default="true">
    <entity schema="public" table="users" x="100" y="200" color="#C1D8EE"></entity>
    <entity schema="public" table="orders" x="400" y="200"></entity>
    <group name="Core Tables" color="#E8F5E9">
      <member schema="public" table="users"></member>
      <member schema="public" table="orders"></member>
    </group>
    <note x="50" y="50" w="200" h="80" color="#FFF9C4">Design notes here</note>
  </layout>
</layouts>
```

### `<layout>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | (required) | Layout name |
| `default` | boolean | `false` | Primary layout shown on open |

### `<entity>` attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `schema` | string | — | Schema name |
| `table` | string | (required) | Table name |
| `x` | int | `0` | X position (pixels) |
| `y` | int | `0` | Y position (pixels) |
| `color` | string | — | Background color (CSS hex: `#RRGGBB`) |
| `width` | int | `0` | Width override (`0` = auto) |
| `height` | int | `0` | Height override (`0` = auto) |

### `<group>` attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Group name |
| `color` | string | Background color |

### `<member>` attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `schema` | string | Schema name |
| `table` | string | Table name |

### `<note>` attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `x` | int | X position |
| `y` | int | Y position |
| `w` | int | Width |
| `h` | int | Height |
| `color` | string | Background color |

Text content is the note body.

---

## Conventions

### Boolean Attributes

Boolean values are represented as `"true"` / `"false"` strings. Omitting a boolean attribute uses its default (documented per element).

### Raw SQL Content

Several elements contain raw SQL as text content, typically wrapped in CDATA sections:
- `<check>` — CHECK expression
- `<expression>` — index expression
- `<where>` — partial index predicate, trigger WHEN
- `<bound>` — partition bound clause
- `<query>` — view/materialized view query
- `<body>` — function body
- `<using>`, `<with-check>` — RLS policy expressions
- `<constraint>` (domain) — CHECK expression
- `<rule>` — rule actions

### Naming

Objects are referenced by name throughout the format (no numeric IDs). Cross-schema references use dot notation (`schema.table`) in the `to-table` and `owned-by` attributes.

### Git-Friendly Design

- Each attribute on a single line for clean diffs
- Predictable element ordering (columns before constraints, schemas before tables)
- No auto-generated IDs that change across saves
- CDATA sections for SQL to avoid XML escaping noise

### Empty Containers

Container elements (`<roles>`, `<extensions>`, `<sequences>`, etc.) may be present but empty, indicating the section exists but has no entries.

---

## Complete Example

```xml
<?xml version="1.0" encoding="UTF-8"?>
<pgd version="1" pg-version="18" default-schema="public">
  <project name="example" description="Example database">
    <settings>
      <naming convention="snake_case" tables="plural"></naming>
      <defaults nullable="true" on-delete="restrict" on-update="restrict"></defaults>
    </settings>
  </project>
  <roles>
    <role name="app_user" login="true"></role>
  </roles>
  <tablespaces></tablespaces>
  <extensions>
    <extension name="uuid-ossp"></extension>
  </extensions>
  <types>
    <enum name="user_status">
      <label>active</label>
      <label>inactive</label>
    </enum>
  </types>
  <sequences>
    <sequence name="order_number_seq" start="1000" increment="1"></sequence>
  </sequences>
  <schema name="public">
    <table name="users" comment="Application users">
      <column name="id" type="bigint" nullable="false">
        <identity generated="always"></identity>
      </column>
      <column name="email" type="varchar" length="255" nullable="false"></column>
      <column name="name" type="text"></column>
      <column name="status" type="user_status" nullable="false" default="'active'"></column>
      <column name="created_at" type="timestamptz" nullable="false" default="now()"></column>
      <pk name="pk_users">
        <column name="id"></column>
      </pk>
      <unique name="uq_users_email">
        <column name="email"></column>
      </unique>
    </table>
    <table name="orders">
      <column name="id" type="bigint" nullable="false">
        <identity generated="always"></identity>
      </column>
      <column name="user_id" type="bigint" nullable="false"></column>
      <column name="order_number" type="bigint" nullable="false"
              default="nextval('order_number_seq')"></column>
      <column name="total" type="numeric" precision="12" scale="2"
              nullable="false" default="0"></column>
      <column name="created_at" type="timestamptz" nullable="false" default="now()"></column>
      <pk name="pk_orders">
        <column name="id"></column>
      </pk>
      <fk name="fk_orders_user" to-table="users" on-delete="cascade" on-update="no action">
        <column name="user_id" references="id"></column>
      </fk>
      <check name="chk_positive_total"><![CDATA[total >= 0]]></check>
    </table>
    <index name="idx_orders_user_id" table="orders">
      <column name="user_id"></column>
    </index>
    <index name="idx_orders_created_at" table="orders">
      <column name="created_at" order="desc"></column>
    </index>
  </schema>
  <views>
    <view name="v_active_users">
      <query><![CDATA[SELECT id, email, name FROM users WHERE status = 'active']]></query>
    </view>
  </views>
  <functions></functions>
  <triggers></triggers>
  <policies></policies>
  <comments>
    <comment on="schema" name="public">Default public schema</comment>
  </comments>
  <grants>
    <grant on="all-tables-in-schema" schema="public" privileges="select" to="app_user"></grant>
  </grants>
  <layouts>
    <layout name="Default Diagram" default="true">
      <entity table="users" x="100" y="100"></entity>
      <entity table="orders" x="400" y="100"></entity>
    </layout>
  </layouts>
</pgd>
```

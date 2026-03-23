package pgd

import "strings"

// pgBuiltinTypes is the set of all PostgreSQL 18 built-in type names,
// including common aliases. Source: docs/architecture/pg18-data-types.md.
var pgBuiltinTypes = map[string]bool{
	// Numeric
	"smallint": true, "integer": true, "bigint": true,
	"int": true, "int2": true, "int4": true, "int8": true,
	"numeric": true, "decimal": true,
	"real": true, "float4": true,
	"double precision": true, "float8": true, "float": true,
	"smallserial": true, "serial": true, "bigserial": true,
	"serial2": true, "serial4": true, "serial8": true,
	"money": true,

	// Character
	"character varying": true, "varchar": true,
	"character": true, "char": true, "bpchar": true,
	"text": true, "name": true,

	// Boolean
	"boolean": true, "bool": true,

	// Date/Time
	"timestamp": true, "timestamp without time zone": true,
	"timestamp with time zone": true, "timestamptz": true,
	"date": true,
	"time": true, "time without time zone": true,
	"time with time zone": true, "timetz": true,
	"interval": true,

	// Binary
	"bytea": true,

	// JSON
	"json": true, "jsonb": true,

	// UUID
	"uuid": true,

	// XML
	"xml": true,

	// Geometric
	"point": true, "line": true, "lseg": true,
	"box": true, "path": true, "polygon": true, "circle": true,

	// Network
	"inet": true, "cidr": true, "macaddr": true, "macaddr8": true,

	// Bit String
	"bit": true, "bit varying": true, "varbit": true,

	// Text Search
	"tsvector": true, "tsquery": true,

	// Range
	"int4range": true, "int8range": true, "numrange": true,
	"tsrange": true, "tstzrange": true, "daterange": true,

	// Multirange
	"int4multirange": true, "int8multirange": true, "nummultirange": true,
	"tsmultirange": true, "tstzmultirange": true, "datemultirange": true,

	// OID / System
	"oid": true, "regclass": true, "regcollation": true,
	"regconfig": true, "regdictionary": true, "regnamespace": true,
	"regoper": true, "regoperator": true, "regproc": true,
	"regprocedure": true, "regrole": true, "regtype": true,
	"xid": true, "xid8": true, "cid": true, "tid": true,
	"pg_lsn": true, "pg_snapshot": true,

	// Pseudo-types (valid as function return types)
	"void": true, "trigger": true, "event_trigger": true,
	"record": true, "cstring": true, "internal": true,
	"anyelement": true, "anyarray": true, "anyrange": true,
	"anymultirange": true, "anyenum": true, "anynonarray": true,
	"anycompatible": true,
}

// StripTypeParams returns the base type name: lowercase, no [], no (params).
func StripTypeParams(typ string) string {
	t := strings.ToLower(strings.TrimSpace(typ))
	t = strings.TrimRight(t, "[]")
	if i := strings.IndexByte(t, '('); i != -1 {
		t = strings.TrimSpace(t[:i])
	}
	return t
}

// IsKnownBuiltinType reports whether typ is a recognised PG18 built-in type.
func IsKnownBuiltinType(typ string) bool {
	return pgBuiltinTypes[StripTypeParams(typ)]
}

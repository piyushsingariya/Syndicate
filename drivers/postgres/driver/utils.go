package driver

import (
	"github.com/piyushsingariya/kaku/types"
)

var pgTypeToDataTypes = map[string]types.DataType{
	// integers
	"bigint":           types.INT64,
	"tinyint":          types.INT64,
	"integer":          types.INT64,
	"smallint":         types.INT64,
	"decimal":          types.FLOAT64,
	"numeric":          types.FLOAT64,
	"double precision": types.FLOAT64,
	"real":             types.FLOAT64,
	"smallserial":      types.INT64,
	"serial":           types.INT64,
	"bigserial":        types.INT64,

	// boolean
	"bool":    types.BOOL,
	"boolean": types.BOOL,

	// strings
	"bit(n)":            types.STRING,
	"varying(n)":        types.STRING,
	"cidr":              types.STRING,
	"inet":              types.STRING,
	"macaddr":           types.STRING,
	"character varying": types.STRING,
	"text":              types.STRING,
	"varchar":           types.STRING,
	"longvarchar":       types.STRING,

	// date/time
	"time":                        types.TIMESTAMP,
	"date":                        types.TIMESTAMP,
	"timestamp":                   types.TIMESTAMP,
	"timestampz":                  types.TIMESTAMP,
	"interval":                    types.INT64,
	"timestamp with time zone":    types.TIMESTAMP,
	"timestamp without time zone": types.TIMESTAMP,

	// arrays
	"ARRAY": types.ARRAY,
	"array": types.ARRAY,

	// binary
	"bytea": types.STRING,
}

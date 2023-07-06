package models

type Schema struct {
	Name string `db:"schema_name"`
}

type Table struct {
	Name string `db:"table_name"`
}

type ColumnDetails struct {
	Name       string  `db:"column_name"`
	DataType   *string `db:"data_type"`
	IsNullable *string `db:"is_nullable"`
}

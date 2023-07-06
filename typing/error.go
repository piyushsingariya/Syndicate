package typing

import (
	"github.com/joomcode/errorx"
)

type ErrorPayload struct {
	Dataset string
	Bucket  string
	Project string

	Database string
	Schema   string
	Table    string

	Cluster         string
	Partition       string
	PrimaryKeys     []string
	Statement       string
	Values          []interface{}
	ValuesMapString string
	TotalObjects    int
}

var (
	// ReportedErrors is an error namespace for reporting errors with codes namespace for general purpose errors designed for universal use.
	reportedErrors = errorx.NewNamespace("report")

	sqlError                  = reportedErrors.NewType("sql")
	BeginTransactionError     = sqlError.NewSubtype("begin_transaction")
	CommitTransactionError    = sqlError.NewSubtype("commit_transaction")
	RollbackTransactionError  = sqlError.NewSubtype("rollback_transaction")
	GetSchemaError            = sqlError.NewSubtype("get_schema")
	CreateSchemaError         = sqlError.NewSubtype("create_schema")
	CreateTableError          = sqlError.NewSubtype("create_table")
	PatchTableError           = sqlError.NewSubtype("patch_table")
	GetTableError             = sqlError.NewSubtype("get_table")
	ReadTableError            = sqlError.NewSubtype("read_table")
	DropError                 = sqlError.NewSubtype("drop_table")
	RenameError               = sqlError.NewSubtype("rename_table")
	CreatePrimaryKeysError    = sqlError.NewSubtype("create_primary_keys")
	DeletePrimaryKeysError    = sqlError.NewSubtype("delete_primary_keys")
	GetPrimaryKeysError       = sqlError.NewSubtype("get_primary_keys")
	DeleteFromTableError      = sqlError.NewSubtype("delete_from_table")
	ExecuteInsertInBatchError = sqlError.NewSubtype("execute_insert_in_batch")
	ExecuteInsertError        = sqlError.NewSubtype("execute_insert")
	UpdateError               = sqlError.NewSubtype("update")
	TruncateError             = sqlError.NewSubtype("truncate")
	BulkMergeError            = sqlError.NewSubtype("bulk_merge")
	CopyError                 = sqlError.NewSubtype("copy")

	stageErr             = reportedErrors.NewType("stage")
	SaveOnStageError     = stageErr.NewSubtype("save_on_stage")
	DeleteFromStageError = stageErr.NewSubtype("delete_from_stage")

	innerError             = reportedErrors.NewType("inner")
	ManageMySQLPrimaryKeys = innerError.NewSubtype("manage_mysql_primary_keys")

	DBInfo          = errorx.RegisterPrintableProperty("db_info")
	DBObjects       = errorx.RegisterPrintableProperty("db_objects")
	SystemErrorFlag = errorx.RegisterPrintableProperty("system_error")

	DestinationID   = errorx.RegisterPrintableProperty("destination_id")
	DestinationType = errorx.RegisterPrintableProperty("destination_type")
)

/*
func Wrap(err error, errorType *errorx.Type, message, propertyKey string, property interface{}) *errorx.Error{
	return errorType.Wrap(err, message).
		WithProperty(propertyKey, property)
}*/

func DecorateError(err error, msg string, args ...interface{}) *errorx.Error {
	return errorx.Decorate(err, msg, args...)
}

func SQLError(errorType *errorx.Type, err error, comment string, payload *ErrorPayload) error {
	return errorType.Wrap(err, comment).
		WithProperty(DBInfo, payload)
}

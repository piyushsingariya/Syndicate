package driver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/piyushsingariya/kaku/drivers/postgres/models"
	"github.com/piyushsingariya/kaku/jsonschema"
	"github.com/piyushsingariya/kaku/jsonschema/schema"
	"github.com/piyushsingariya/kaku/logger"
	kakumodels "github.com/piyushsingariya/kaku/models"
	"github.com/piyushsingariya/kaku/protocol"
	"github.com/piyushsingariya/kaku/types"
	"github.com/piyushsingariya/kaku/typing"
	"github.com/piyushsingariya/kaku/utils"
)

const (
	// get all schemas
	getSchemaNamesTmpl = `SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT LIKE 'pg_%' AND schema_name != 'information_schema'`
	// get all tables from a schema
	getSchemaTablesTmpl = `SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type = 'BASE TABLE'`
	// get table schema
	getTableSchemaTmpl = `SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`
	// get primary key columns
	getTablePrimaryKey = `SELECT column_name FROM information_schema.key_column_usage WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`
)

type Postgres struct {
	allStreams  map[string]*pgStream
	batchSize   int64
	client      *sqlx.DB
	accessToken string
	config      *models.Config
	catalog     *kakumodels.Catalog
	state       kakumodels.State
}

func (p *Postgres) Setup(config any, catalog *kakumodels.Catalog, state kakumodels.State, batchSize int64) error {
	cfg := models.Config{}
	err := utils.Unmarshal(config, &cfg)
	if err != nil {
		return err
	}

	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	db, err := sqlx.Open("pgx", cfg.ToConnectionString())
	if err != nil {
		return fmt.Errorf("failed to connect database: %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// force a connection and test that it worked
	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping database: %s", err)
	}

	p.client = db.Unsafe()

	p.config = &cfg
	p.catalog = catalog
	p.state = state
	p.batchSize = batchSize

	return p.setupStreams()
}

func (p *Postgres) CloseConnection() {
	if p.client != nil {
		err := p.client.Close()
		if err != nil {
			logger.Error("failed to close connection with postgres: %s", err)
		}
	}
}

func (p *Postgres) Spec() (schema.JSONSchema, error) {
	return jsonschema.Reflect(models.Config{})
}

func (p *Postgres) Check() error {
	return nil
}

func (p *Postgres) Discover() ([]*kakumodels.Stream, error) {
	streams := []*kakumodels.Stream{}
	for _, stream := range p.allStreams {
		streams = append(streams, stream.Stream)
	}

	return streams, nil
}

func (p *Postgres) Catalog() *kakumodels.Catalog {
	return p.catalog
}
func (p *Postgres) Type() string {
	return "Postgres"
}

func (p *Postgres) Streams() ([]*kakumodels.Stream, error) {
	return nil, nil
}
func (p *Postgres) Read(stream protocol.Stream, channel chan<- kakumodels.Record) error {
	identifier := utils.StreamIdentifier(stream.Namespace(), stream.Name())
	pgStream, found := p.allStreams[identifier]
	if !found {
		logger.Warnf("Stream %s.%s not found; skipping...", stream.Namespace(), stream.Name())
		return nil
	}

	if !utils.ArrayContains(pgStream.SupportedSyncModes, stream.GetSyncMode()) {
		logger.Warnf("Stream %s.%s does not support sync mode[%s]; skipping...", stream.Namespace(), stream.Name(), stream.GetSyncMode())
		return nil
	}

	switch stream.GetSyncMode() {
	case types.FullRefresh:
		return pgStream.readFullRefresh(p.client, channel)
	case types.Incremental:
		// check if cursor field is supported
		if !utils.ArrayContains(pgStream.DefaultCursorFields, stream.GetCursorField()) {
			logger.Warnf("Stream %s.%s does not support cursor field[%s]; skipping...", stream.Namespace(), stream.Name(), stream.GetCursorField())
			return nil
		}

		// set state
		pgStream.setState(stream.GetCursorField(), p.state.Get(stream.Name(), stream.Namespace())[stream.GetCursorField()])

		// read incrementally
		return pgStream.readIncremental(p.client, channel)
	}

	return nil
}

func (p *Postgres) GetState() (*kakumodels.State, error) {
	state := &kakumodels.State{}
	for _, stream := range p.Catalog().Streams {
		if stream.SyncMode == types.Incremental || stream.SyncMode == types.CDC {
			pgStream, found := p.allStreams[utils.StreamIdentifier(stream.Namespace(), stream.Name())]
			if !found {
				return nil, fmt.Errorf("postgres stream not found while getting state of stream %s[%s]", stream.Name(), stream.Namespace())
			}

			if !(utils.ArrayContains(pgStream.SupportedSyncModes, types.Incremental) || utils.ArrayContains(pgStream.SupportedSyncModes, types.CDC)) {
				logger.Warnf("Skipping getting state from stream %s[%s], this stream doesn't support incremental/CDC", stream.Name(), stream.Namespace())
				continue
			}

			state.Add(stream.Name(), stream.Namespace(), map[string]any{
				pgStream.cursor: pgStream.state,
			})
		}
	}

	return state, nil
}

func (p *Postgres) setupStreams() error {
	p.allStreams = make(map[string]*pgStream)

	var schemaNamesOutput []models.Schema
	err := p.client.Select(&schemaNamesOutput, getSchemaNamesTmpl)
	if err != nil {
		return fmt.Errorf("failed to get schema names: %s", err)
	}

	if len(schemaNamesOutput) == 0 {
		return typing.SQLError(typing.GetSchemaError, err, "no schemas found in database", &typing.ErrorPayload{
			Statement: getSchemaTablesTmpl,
		})
	}

	for _, schema := range schemaNamesOutput {
		var tableNamesOutput []models.Table
		err := p.client.Select(&tableNamesOutput, getSchemaTablesTmpl, schema.Name)
		if err != nil {
			return fmt.Errorf("failed to retrieve table names from schema[%s]: %s", schema.Name, err)
		}

		if len(tableNamesOutput) == 0 {
			logger.Warnf("no tables found in schema[%s]", schema.Name)
		}

		for _, table := range tableNamesOutput {
			var columnSchemaOutput []models.ColumnDetails
			err := p.client.Select(&columnSchemaOutput, getTableSchemaTmpl, schema.Name, table.Name)
			if err != nil {
				return fmt.Errorf("failed to retrieve column details for table %s[%s]: %s", table.Name, schema.Name, err)
			}

			if len(columnSchemaOutput) == 0 {
				logger.Warnf("no columns found in table %s[%s]", table.Name, schema.Name)
				continue
			}

			var primaryKeyOutput []models.ColumnDetails
			err = p.client.Select(&primaryKeyOutput, getTablePrimaryKey, schema.Name, table.Name)
			if err != nil {
				return fmt.Errorf("failed to retrieve primary key columns for table %s[%s]: %s", table.Name, schema.Name, err)
			}

			stream := &kakumodels.Stream{
				Name:      table.Name,
				Namespace: schema.Name,
			}

			stream.JSONSchema = &kakumodels.Schema{
				Properties: make(map[string]*kakumodels.Property),
			}

			for _, column := range columnSchemaOutput {
				datatypes := []types.DataType{}
				if val, found := pgTypeToDataTypes[*column.DataType]; found {
					datatypes = append(datatypes, val)
				} else {
					logger.Warnf("failed to get respective type in datatypes for column: %s[%s]", column.Name, column.DataType)
					datatypes = append(datatypes, types.UNKNOWN)
				}

				if strings.EqualFold("yes", *column.IsNullable) {
					datatypes = append(datatypes, types.NULL)
				}

				stream.JSONSchema.Properties[column.Name] = &kakumodels.Property{
					Type: datatypes,
				}
			}

			stream.SupportedSyncModes = append(stream.SupportedSyncModes, types.FullRefresh)

			// currently only datetime fields is supported for cursor field, automatic generated fields can also be used
			// future TODO
			for propertyName, property := range stream.JSONSchema.Properties {
				if utils.ArrayContains(property.Type, types.TIMESTAMP) {
					stream.DefaultCursorFields = append(stream.DefaultCursorFields, propertyName)
				}
			}

			// source has cursor fields, hence incremental also supported
			if len(stream.DefaultCursorFields) > 0 {
				stream.SourceDefinedCursor = true
				stream.SupportedSyncModes = append(stream.SupportedSyncModes, types.Incremental)
			}

			// add primary keys for stream
			for _, column := range primaryKeyOutput {
				stream.SourceDefinedPrimaryKey = append(stream.SourceDefinedPrimaryKey, column.Name)
			}

			p.allStreams[utils.StreamIdentifier(schema.Name, table.Name)] = &pgStream{
				Stream:    stream,
				batchSize: p.batchSize,
			}
		}

	}

	return nil
}

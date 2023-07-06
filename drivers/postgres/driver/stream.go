package driver

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	kakumodels "github.com/piyushsingariya/kaku/models"
	"github.com/piyushsingariya/kaku/safego"
	"github.com/piyushsingariya/kaku/types"
	"github.com/piyushsingariya/kaku/typing"
	"github.com/piyushsingariya/kaku/utils"
)

type pgStream struct {
	batchSize int64
	cursor    string
	state     interface{}
	*kakumodels.Stream
}

const (
	readRecordsFullRefresh             = `SELECT * FROM "%s"."%s" OFFSET %d LIMIT %d`
	readRecordsIncrementalWithState    = `SELECT * FROM "%s"."%s" where "%s">= $1 ORDER BY "%s" ASC OFFSET %d LIMIT %d`
	readRecordsIncrementalWithoutState = `SELECT * FROM "%s"."%s" ORDER BY "%s" ASC OFFSET %d LIMIT %d`
)

func (p *pgStream) setState(cursor string, state interface{}) {
	p.cursor = cursor
	p.state = state
}

func (p *pgStream) readFullRefresh(client *sqlx.DB, channel chan<- kakumodels.Record) error {
	offset := int64(0)
	limit := p.batchSize

	for {
		statement := fmt.Sprintf(readRecordsFullRefresh, p.Namespace, p.Name, offset*limit, limit)

		// Execute the query
		rows, err := client.Queryx(statement)
		if err != nil {
			return typing.SQLError(typing.ReadTableError, err, fmt.Sprintf("failed to read after offset[%d] limit[%d]", offset*limit, limit), &typing.ErrorPayload{
				Table:     p.Name,
				Schema:    p.Namespace,
				Statement: statement,
			})
		}

		paginationFinished := true

		// Fetch rows and populate the result
		for rows.Next() {
			paginationFinished = false

			// Create a map to hold column names and values
			record := make(types.RecordData)

			// Scan the row into the map
			err := rows.MapScan(record)
			if err != nil {
				return fmt.Errorf("failed to mapScan record data: %s", err)
			}

			// insert record
			if !safego.Insert(channel, utils.ReformatRecord(p.Name, p.Namespace, record)) {
				// channel was closed
				return nil
			}
		}

		// Check for any errors during row iteration
		err = rows.Err()
		if err != nil {
			return fmt.Errorf("failed to mapScan record data: %s", err)
		}

		// records finished
		if paginationFinished {
			break
		}

		// increase offset
		offset += 1
		rows.Close()
	}
	return nil
}

func (p *pgStream) readIncremental(client *sqlx.DB, channel chan<- kakumodels.Record) error {
	offset := int64(0)
	limit := p.batchSize
	var initialStateAtStart any
	if p.state != nil {
		initialStateAtStart = p.state
	}

	var extract func() (*sqlx.Rows, error) = func() (*sqlx.Rows, error) {
		if initialStateAtStart != nil {
			statement := fmt.Sprintf(readRecordsIncrementalWithState, p.Namespace, p.Name, p.cursor, p.cursor, offset*limit, limit)
			// Execute the query
			return client.Queryx(statement, initialStateAtStart)
		}
		statement := fmt.Sprintf(readRecordsIncrementalWithoutState, p.Namespace, p.Name, p.cursor, offset*limit, limit)
		// Execute the query
		return client.Queryx(statement)
	}

	for {
		// extract rows
		rows, err := extract()
		if err != nil {
			return typing.SQLError(typing.ReadTableError, err, fmt.Sprintf("failed to read after offset[%d] limit[%d]", offset*limit, limit), &typing.ErrorPayload{
				Table:  p.Name,
				Schema: p.Namespace,
			})
		}

		paginationFinished := true

		// Fetch rows and populate the result
		for rows.Next() {
			paginationFinished = false

			// Create a map to hold column names and values
			record := make(types.RecordData)

			// Scan the row into the map
			err := rows.MapScan(record)
			if err != nil {
				return fmt.Errorf("failed to mapScan record data: %s", err)
			}

			if cursorVal, found := record[p.cursor]; found && cursorVal != nil {
				// compare if not nil
				if p.state != nil {
					state, err := utils.MaximumOnDataType(p.JSONSchema.Properties[p.cursor].Type, p.state, cursorVal)
					if err != nil {
						return err
					}

					p.state = state
				} else {
					// directly update
					p.state = cursorVal
				}
			}

			// insert record
			channel <- utils.ReformatRecord(p.Name, p.Namespace, record)
		}

		// Check for any errors during row iteration
		err = rows.Err()
		if err != nil {
			return fmt.Errorf("failed to mapScan record data: %s", err)
		}

		// records finished
		if paginationFinished {
			break
		}

		// increase offset
		offset += 1
		rows.Close()
	}

	return nil
}

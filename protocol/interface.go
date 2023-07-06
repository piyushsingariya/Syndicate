package protocol

import (
	"github.com/piyushsingariya/kaku/jsonschema/schema"
	"github.com/piyushsingariya/kaku/models"
	"github.com/piyushsingariya/kaku/types"
)

type Connector interface {
	Setup(config any, catalog *models.Catalog, state models.State, batchSize int64) error
	Spec() (schema.JSONSchema, error)
	Check() error
	Discover() ([]*models.Stream, error)

	Catalog() *models.Catalog
	Type() string
}

type Driver interface {
	Connector
	Streams() ([]*models.Stream, error)
	Read(stream Stream, channel chan<- models.Record) error
	GetState() (*models.State, error)
}

type Adapter interface {
	Connector
	Write(channel <-chan models.Record) error
	Create(streamName string) error
}

type Stream interface {
	Name() string
	Namespace() string
	JSONSchema() *models.Schema
	GetStream() *models.Stream
	GetSyncMode() types.SyncMode
	GetCursorField() string
}

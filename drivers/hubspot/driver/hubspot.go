package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/piyushsingariya/kaku/drivers/hubspot/models"
	"github.com/piyushsingariya/kaku/jsonschema"
	"github.com/piyushsingariya/kaku/jsonschema/schema"
	"github.com/piyushsingariya/kaku/logger"
	kakumodels "github.com/piyushsingariya/kaku/models"
	"github.com/piyushsingariya/kaku/protocol"
	"github.com/piyushsingariya/kaku/types"
	"github.com/piyushsingariya/kaku/utils"
)

type Hubspot struct {
	batchSize   int64
	allStreams  map[string]HubspotStream
	client      *http.Client
	accessToken string
	config      *models.Config
	catalog     *kakumodels.Catalog
	state       kakumodels.State
}

func (h *Hubspot) Setup(config any, catalog *kakumodels.Catalog, state kakumodels.State, batchSize int64) error {
	conf := &models.Config{}
	if err := utils.Unmarshal(config, conf); err != nil {
		return err
	}

	if err := conf.ValidateAndPopulateDefaults(); err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	client, accessToken, err := newClient(conf)
	if err != nil {
		return err
	}

	h.catalog = catalog
	h.config = conf
	h.state = state
	h.batchSize = batchSize
	h.client = client
	h.accessToken = accessToken
	h.setupAllStreams()

	return nil
}

func (h *Hubspot) Spec() (schema.JSONSchema, error) {
	return jsonschema.Reflect(models.Config{})
}

func (h *Hubspot) Check() error {
	return nil
}

func (h *Hubspot) Discover() ([]*kakumodels.Stream, error) {
	streams := []*kakumodels.Stream{}

	for streamName, hstream := range h.allStreams {
		stream := &kakumodels.Stream{
			Name:                    streamName,
			SupportedSyncModes:      hstream.Modes(),
			SourceDefinedPrimaryKey: hstream.PrimaryKey(),
		}

		if hstream.cursorField() != "" {
			stream.DefaultCursorFields = append(stream.DefaultCursorFields, hstream.cursorField())
			stream.SourceDefinedCursor = true
		}

		streams = append(streams, stream)
	}

	return streams, nil
}

func (h *Hubspot) Catalog() *kakumodels.Catalog {
	return h.catalog
}
func (h *Hubspot) Type() string {
	return "Hubspot"
}

func (h *Hubspot) Streams() ([]*kakumodels.Stream, error) {
	scopes, err := h.getGrantedScopes()
	if err != nil {
		return nil, err
	}

	logger.Infof("The following scopes are granted: %v", scopes)
	return nil, nil
}

func (h *Hubspot) GetState() (*kakumodels.State, error) {
	state := &kakumodels.State{}
	for _, stream := range h.Catalog().Streams {
		if stream.SyncMode == types.Incremental {
			hubspotStream, found := h.allStreams[stream.Name()]
			if !found {
				return nil, fmt.Errorf("hubspot stream not found while getting state of incremental stream[%s]", stream.Name())
			}

			if !utils.ArrayContains(hubspotStream.Modes(), types.Incremental) {
				logger.Warnf("Skipping getting state from stream[%s], this stream doesn't support incremental", stream.Name())
				continue
			}

			state.Add(stream.Name(), stream.Name(), hubspotStream.state())
		}
	}

	return state, nil
}

func (h *Hubspot) Read(stream protocol.Stream, channel chan<- kakumodels.Record) error {
	hstream, found := h.allStreams[stream.Name()]
	if !found {
		return fmt.Errorf("invalid stream passed: %s", stream.Name())
	}

	hstream.setup(stream.GetSyncMode(), h.state.Get(stream.Name(), stream.Namespace()))

	err := hstream.readRecords(channel)
	if err != nil {
		return fmt.Errorf("error occurred: %s", err)
	}

	return nil
}

func (h *Hubspot) getGrantedScopes() ([]string, error) {
	req, err := http.NewRequest("GET", formatEndpoint(fmt.Sprintf("oauth/v1/access-tokens/%s", h.accessToken)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := map[string]any{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response["scopes"].([]string), nil
}

func (h *Hubspot) register(streamName string, stream HubspotStream) {
	h.allStreams[streamName] = stream
}

func (h *Hubspot) setupAllStreams() {
	// initialization
	h.allStreams = map[string]HubspotStream{}

	h.register("contacts",
		newCRMSearchStream(*newIncrementalStream("contacts", "contact", h.client, h.config.StartDate),
			"id", "lastmodifieddate",
			[]string{"contacts", "companies"}, []string{"crm.objects.contacts.read"}))

	h.register("companies",
		newCRMSearchStream(
			*newIncrementalStream("companies", "company", h.client, h.config.StartDate),
			"id", "hs_lastmodifieddate",
			[]string{"contacts"}, []string{"crm.objects.contacts.read", "crm.objects.companies.read"}))

	h.register("engagementscalls",
		newCRMSearchStream(
			*newIncrementalStream("engagementscalls", "calls", h.client, h.config.StartDate),
			"id", "hs_lastmodifieddate",
			[]string{"contacts", "deal", "company", "tickets"}, []string{"crm.objects.contacts.read"}))

}

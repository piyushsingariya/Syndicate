package protocol

import (
	"fmt"
	"strings"
	"time"

	"github.com/piyushsingariya/kaku/logger"
	"github.com/piyushsingariya/kaku/models"
	"github.com/piyushsingariya/kaku/utils"
	"github.com/spf13/cobra"
)

// ReadCmd represents the read command
var ReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Kaku read command",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := utils.CheckIfFilesExists(config, catalog); err != nil {
			return err
		}

		if state != "" {
			return utils.CheckIfFilesExists(state)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		connector, not := rawConnector.(Driver)
		if !not {
			return fmt.Errorf("expected type to be: Driver, found %T", connector)
		}

		cat := &models.Catalog{}
		if err := utils.Unmarshal(utils.ReadFile(catalog), cat); err != nil {
			return fmt.Errorf("failed to unmarshal catalog:%s", err)
		}

		if state == "" {
			if err := connector.Setup(utils.ReadFile(config), cat, nil, batchSize); err != nil {
				return err
			}
		} else {
			st := models.State{}
			err := utils.Unmarshal(utils.ReadFile(state), &st)
			if err != nil {
				return fmt.Errorf("failed to unmarshal state file")
			}
			if err := connector.Setup(utils.ReadFile(config), cat, st, batchSize); err != nil {
				return err
			}
		}

		recordStream := make(chan models.Record, 2*batchSize)
		numRecords := int64(0)
		batch := int64(0)

		go func() {
			for message := range recordStream {
				logger.LogRecord(message)
				numRecords++
				batch++

				// log state after a batch
				if batch >= batchSize {
					state, err := connector.GetState()
					if err != nil {
						logger.Fatalf("failed to get state from connector")
					}
					if state != nil && state.Len() > 0 {
						logger.LogState(state)
					}
					// reset batch
					batch = 0
				}
			}
		}()

		streamNames := []string{}

		for _, stream := range connector.Catalog().Streams {
			if stream.Namespace() != "" {
				streamNames = append(streamNames, fmt.Sprintf("%s[%s]", stream.Name(), stream.Namespace()))
			} else {
				streamNames = append(streamNames, stream.Name())
			}
		}
		logger.Infof("Selected streams are %s", strings.Join(streamNames, ", "))

		for _, stream := range connector.Catalog().Streams {
			if stream.Namespace() != "" {
				logger.Infof("Reading stream %s[%s]", stream.Name(), stream.Namespace())
			} else {
				logger.Infof("Reading stream %s", stream.Name())
			}

			streamStartTime := time.Now()
			err := connector.Read(stream, recordStream)
			if err != nil {
				logger.Fatalf("Error occurred while reading records from [%s]: %s", connector.Type(), err)
			}

			logger.Infof("Finished reading stream %s[%s] in %s", stream.Name(), stream.Namespace(), time.Since(streamStartTime).String())
		}

		close(recordStream)

		logger.Infof("Total records read: %d", numRecords)
		state, err := connector.GetState()
		if err != nil {
			logger.Fatalf("failed to get state from connector")
		}

		if state != nil {
			logger.LogState(state)
		}

		return nil
	},
}

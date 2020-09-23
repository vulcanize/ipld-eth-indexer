// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package historical

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/params"
	log "github.com/sirupsen/logrus"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
	"github.com/vulcanize/ipld-eth-indexer/utils"
)

// Backfill for filling in gaps in the ipld-eth-indexer db
type Backfill interface {
	// Method for the watcher to periodically check for and fill in gaps in its data using an archival node
	Sync(wg *sync.WaitGroup)
	Stop() error
}

// Service for filling in gaps in the watcher
type Service struct {
	// Interface for fetching statediff.Payloads over http
	Fetcher eth.Fetcher
	// Interface for transforming payloads into IPLD object models in Postgres
	Transformer eth.Transformer
	// Interface for finding gaps in the database
	Retriever eth.Retriever
	// Check frequency
	GapCheckFrequency time.Duration
	// Size of batch fetches
	BatchSize uint64
	// Number of goroutines
	Workers int64
	// Channel for receiving quit signal
	QuitChan chan bool
	// Chain config
	ChainConfig *params.ChainConfig
	// Headers with times_validated lower than this will be resynced
	validationLevel int
}

// NewBackfillService returns a new BackfillInterface
func NewBackfillService(settings *Config) (Backfill, error) {
	bs := new(Service)
	var err error
	bs.Fetcher = eth.NewPayloadFetcher(settings.HTTPClient, settings.Timeout)
	bs.ChainConfig, err = eth.ChainConfig(settings.NodeInfo.ChainID)
	if err != nil {
		return nil, err
	}
	bs.Transformer = eth.NewStateDiffTransformer(bs.ChainConfig, settings.DB)
	bs.Retriever = eth.NewGapRetriever(settings.DB)
	bs.BatchSize = settings.BatchSize
	if bs.BatchSize == 0 {
		bs.BatchSize = shared.DefaultMaxBatchSize
	}
	bs.Workers = int64(settings.Workers)
	if bs.Workers == 0 {
		bs.Workers = shared.DefaultMaxBatchNumber
	}
	bs.QuitChan = make(chan bool)
	bs.validationLevel = settings.ValidationLevel
	bs.GapCheckFrequency = settings.Frequency
	return bs, nil
}

// Sync periodically checks for and fills in gaps in the watcher db
func (bfs *Service) Sync(wg *sync.WaitGroup) {
	ticker := time.NewTicker(bfs.GapCheckFrequency)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-bfs.QuitChan:
				log.Info("quiting ethereum backfill process")
				return
			case <-ticker.C:
				gaps, err := bfs.Retriever.RetrieveGapsInData(bfs.validationLevel)
				if err != nil {
					log.Errorf("ethereum backfill error finding missing data: %v", err)
					continue
				}
				// spin up worker goroutines for this search pass
				// we start and kill a new batch of workers for each pass
				// so that we know each of the previous workers is done before we search for new gaps
				heightsChan := make(chan []uint64)
				for i := 1; i <= int(bfs.Workers); i++ {
					go bfs.backFill(wg, i, heightsChan)
				}
				for _, gap := range gaps {
					log.Infof("backfilling historical ethereum data from %d to %d", gap.Start, gap.Stop)
					blockRangeBins, err := utils.GetBlockHeightBins(gap.Start, gap.Stop, bfs.BatchSize)
					if err != nil {
						log.Errorf("ethereum backfill gap binning error: %v", err)
						continue
					}
					for _, heights := range blockRangeBins {
						select {
						case <-bfs.QuitChan:
							log.Info("quiting ethereum backfill process")
							return
						default:
							heightsChan <- heights
						}
					}
				}
				// send a quit signal to each worker
				// this blocks until each worker has finished its current task and is free to receive from the quit channel
				for i := 1; i <= int(bfs.Workers); i++ {
					bfs.QuitChan <- true
				}
			}
		}
	}()
	log.Info("ethereum backfill process successfully spun up")
}

func (bfs *Service) backFill(wg *sync.WaitGroup, id int, heightChan chan []uint64) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case heights := <-heightChan:
			log.Debugf("ethereum backfill worker %d processing section from %d to %d", id, heights[0], heights[len(heights)-1])
			payloads, err := bfs.Fetcher.FetchAt(heights)
			if err != nil {
				log.Errorf("ethereum backfill worker %d fetcher error: %s", id, err.Error())
			}
			for _, payload := range payloads {
				blockNumber, err := bfs.Transformer.Transform(id, payload)
				if err != nil {
					log.Errorf("ethereum backfill worker %d transformer error: %s", id, err.Error())
				}
				log.Infof("ethereum backfill worker %d transformed data at height %d", id, blockNumber)
			}
			log.Infof("ethereum backfill worker %d finished section from %d to %d", id, heights[0], heights[len(heights)-1])
		case <-bfs.QuitChan:
			log.Infof("ethereum backfill worker %d shutting down", id)
			return
		}
	}
}

func (bfs *Service) Stop() error {
	log.Info("stopping ethereum backfill service")
	close(bfs.QuitChan)
	return nil
}

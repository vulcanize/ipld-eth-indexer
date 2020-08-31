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

package resync

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
	"github.com/sirupsen/logrus"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
	"github.com/vulcanize/ipld-eth-indexer/utils"
)

type Resync interface {
	Sync() error
}

type Service struct {
	// Interface for fetching historical statediff objects over http
	Fetcher eth.Fetcher
	// Interface for converting payloads into IPLD object payloads
	Converter eth.Converter
	// Interface for publishing the IPLD payloads to IPFS
	Publisher eth.Publisher
	// Interface for cleaning out data before resyncing (if clearOldCache is on)
	Cleaner eth.Cleaner
	// Size of batch fetches
	BatchSize uint64
	// Number of goroutines
	Workers int64
	// Channel for receiving quit signal
	quitChan chan bool
	// Chain config
	ChainConfig *params.ChainConfig
	// Resync data type
	data shared.DataType
	// Resync ranges
	ranges [][2]uint64
	// Flag to turn on or off old cache destruction
	clearOldCache bool
	// Flag to turn on or off validation level reset
	resetValidation bool
}

// NewResyncService creates and returns a resync service from the provided settings
func NewResyncService(settings *Config) (Resync, error) {
	rs := new(Service)
	var err error
	rs.Fetcher = eth.NewPayloadFetcher(settings.HTTPClient, settings.Timeout)
	rs.ChainConfig, err = eth.ChainConfig(settings.NodeInfo.ChainID)
	if err != nil {
		return nil, err
	}
	rs.Converter = eth.NewPayloadConverter(rs.ChainConfig)
	rs.Publisher = eth.NewIPLDPublisher(settings.DB)
	rs.Cleaner = eth.NewDBCleaner(settings.DB)
	rs.BatchSize = settings.BatchSize
	if rs.BatchSize == 0 {
		rs.BatchSize = shared.DefaultMaxBatchSize
	}
	rs.Workers = int64(settings.Workers)
	if rs.Workers == 0 {
		rs.Workers = shared.DefaultMaxBatchNumber
	}
	rs.resetValidation = settings.ResetValidation
	rs.clearOldCache = settings.ClearOldCache
	rs.quitChan = make(chan bool)
	rs.ranges = settings.Ranges
	rs.data = settings.ResyncType
	return rs, nil
}

// Sync indexes data within a specified block range
func (rs *Service) Sync() error {
	if rs.resetValidation {
		logrus.Infof("resetting validation level")
		if err := rs.Cleaner.ResetValidation(rs.ranges); err != nil {
			return fmt.Errorf("validation reset failed: %v", err)
		}
	}
	if rs.clearOldCache {
		logrus.Infof("cleaning out old data from Postgres")
		if err := rs.Cleaner.Clean(rs.ranges, rs.data); err != nil {
			return fmt.Errorf("ethereum %s data resync cleaning error: %v", rs.data.String(), err)
		}
	}
	// spin up worker goroutines
	heightsChan := make(chan []uint64)
	for i := 1; i <= int(rs.Workers); i++ {
		go rs.resync(i, heightsChan)
	}
	for _, rng := range rs.ranges {
		if rng[1] < rng[0] {
			logrus.Error("ethereum resync range ending block number needs to be greater than the starting block number")
			continue
		}
		logrus.Infof("resyncing ethereum data from %d to %d", rng[0], rng[1])
		// break the range up into bins of smaller ranges
		blockRangeBins, err := utils.GetBlockHeightBins(rng[0], rng[1], rs.BatchSize)
		if err != nil {
			return err
		}
		for _, heights := range blockRangeBins {
			heightsChan <- heights
		}
	}
	// send a quit signal to each worker
	// this blocks until each worker has finished its current task and can receive from the quit channel
	for i := 1; i <= int(rs.Workers); i++ {
		rs.quitChan <- true
	}
	return nil
}

func (rs *Service) resync(id int, heightChan chan []uint64) {
	for {
		select {
		case heights := <-heightChan:
			logrus.Debugf("ethereum resync worker %d processing section from %d to %d", id, heights[0], heights[len(heights)-1])
			payloads, err := rs.Fetcher.FetchAt(heights)
			if err != nil {
				logrus.Errorf("ethereum resync worker %d fetcher error: %s", id, err.Error())
			}
			for _, payload := range payloads {
				ipldPayload, err := rs.Converter.Convert(payload)
				if err != nil {
					logrus.Errorf("ethereum resync worker %d converter error: %s", id, err.Error())
				}
				if err := rs.Publisher.Publish(*ipldPayload); err != nil {
					logrus.Errorf("ethereum resync worker %d publisher error: %s", id, err.Error())
				}
			}
			logrus.Infof("ethereum resync worker %d finished section from %d to %d", id, heights[0], heights[len(heights)-1])
		case <-rs.quitChan:
			logrus.Infof("ethereum resync worker %d goroutine shutting down", id)
			return
		}
	}
}

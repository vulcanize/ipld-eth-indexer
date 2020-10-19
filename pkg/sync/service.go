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

package sync

import (
	"sync"

	ethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	log "github.com/sirupsen/logrus"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/prom"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
)

const (
	PayloadChanBufferSize = 2000
)

// Indexer is the top level interface for streaming, converting to IPLDs, publishing, and indexing all chain data at head
// This service is compatible with the Ethereum service interface (node.Service)
type Indexer interface {
	// APIs(), Protocols(), Start() and Stop()
	ethnode.Service
	// Data processing event loop
	Sync(wg *sync.WaitGroup) error
	// Method to access chain type
	Chain() shared.ChainType
}

// Service is the underlying struct for the indexer
type Service struct {
	// Interface for streaming payloads over an rpc subscription
	Streamer eth.Streamer
	// Interface for transforming raw payloads into IPLD object models in Postgres
	Transformer eth.Transformer
	// Chan the processor uses to subscribe to payloads from the Streamer
	PayloadChan chan statediff.Payload
	// Used to signal shutdown of the service
	QuitChan chan bool
	// Number of sync workers
	Workers int64
	// chain type for this service
	ChainConfig *params.ChainConfig
}

// NewIndexer creates a new Indexer using an underlying Service struct
func NewIndexerService(settings *Config) (Indexer, error) {
	sn := new(Service)
	var err error
	sn.PayloadChan = make(chan statediff.Payload, eth.PayloadChanBufferSize)
	sn.Streamer = eth.NewPayloadStreamer(settings.WSClient)
	sn.ChainConfig, err = eth.ChainConfig(settings.NodeInfo.ChainID)
	if err != nil {
		return nil, err
	}
	sn.Transformer = eth.NewStateDiffTransformer(sn.ChainConfig, settings.DB)
	sn.QuitChan = make(chan bool)
	sn.Workers = settings.Workers
	return sn, nil
}

// Protocols exports the services p2p protocols, this service has none
func (sap *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns the RPC descriptors the indexer service offers
func (sap *Service) APIs() []rpc.API {
	return []rpc.API{}
}

// Sync streams incoming raw chain data and converts it for further processing
// It forwards the converted data to the publish process(es) it spins up
// This continues on no matter if or how many subscribers there are
func (sap *Service) Sync(wg *sync.WaitGroup) error {
	sub, err := sap.Streamer.Stream(sap.PayloadChan)
	if err != nil {
		return err
	}
	// spin up publish worker goroutines
	publishPayload := make(chan statediff.Payload, PayloadChanBufferSize)
	for i := 1; i <= int(sap.Workers); i++ {
		go sap.transform(wg, i, publishPayload)
		log.Debugf("ethereum sync worker %d successfully spun up", i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case diffPayload := <-sap.PayloadChan:
				select {
				case publishPayload <- diffPayload:
				default:
					<-publishPayload
					publishPayload <- diffPayload
				}
				prom.SetLenPayloadChan(len(publishPayload))
			case err := <-sub.Err():
				log.Errorf("ethereumm sync subscription error: %v", err)
			case <-sap.QuitChan:
				log.Info("quiting ethereum sync process")
				return
			}
		}
	}()
	log.Info("ethereum sync process successfully spun up")
	return nil
}

// transform is spun up by Sync and receives statediff payloads from it
// it transforms this data into IPLD models and indexes their CIDs with useful metadata in Postgres
func (sap *Service) transform(wg *sync.WaitGroup, id int, statediffChan <-chan statediff.Payload) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case diff := <-statediffChan:
			prom.SetLenPayloadChan(len(statediffChan))
			blockNumber, err := sap.Transformer.Transform(id, diff)
			if err != nil {
				log.Errorf("ethereum sync worker %d transformer error: %v", id, err)
			}
			log.Infof("ethereum sync worker %d transformed data at height %d", id, blockNumber)
		case <-sap.QuitChan:
			log.Infof("ethereum sync worker %d shutting down", id)
			return
		}
	}
}

// Start is used to begin the service
// This is mostly just to satisfy the node.Service interface
func (sap *Service) Start(*p2p.Server) error {
	log.Info("starting ethereum indexer service")
	wg := new(sync.WaitGroup)
	return sap.Sync(wg)
}

// Stop is used to close down the service
// This is mostly just to satisfy the node.Service interface
func (sap *Service) Stop() error {
	log.Info("stopping ethereum indexer service")
	close(sap.QuitChan)
	return nil
}

// Chain returns the chain type for this service
func (sap *Service) Chain() shared.ChainType {
	return shared.Ethereum
}

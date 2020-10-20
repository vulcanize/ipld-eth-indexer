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

package mocks

import (
	"github.com/ethereum/go-ethereum/statediff"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
)

// PayloadStreamer mock struct
type PayloadStreamer struct {
	PassedPayloadChan  chan statediff.Payload
	PassedCodeChan     chan statediff.CodeAndCodeHash
	ReturnErr          error
	StreamPayloads     []statediff.Payload
	StreamCodePayloads map[uint64][]statediff.CodeAndCodeHash
}

// Stream mock method
func (sds *PayloadStreamer) Stream(payloadChan chan statediff.Payload) (eth.ClientSubscription, error) {
	sds.PassedPayloadChan = payloadChan

	errChan := make(chan error)
	quitChan := make(chan bool)
	clientSub := &ClientSubscription{
		errChan: errChan,
	}
	go func() {
		for _, payload := range sds.StreamPayloads {
			select {
			case <-quitChan:
				return
			default:
			}
			sds.PassedPayloadChan <- payload
		}
		close(errChan)
	}()

	return clientSub, sds.ReturnErr
}

// StreamCodeAndCodeHash mock method
func (sds *PayloadStreamer) StreamCodeAndCodeHash(payloadChan chan statediff.CodeAndCodeHash, blockNumber uint64) (eth.ClientSubscription, error) {
	sds.PassedCodeChan = payloadChan
	if sds.StreamCodePayloads == nil {
		return nil, nil
	}

	errChan := make(chan error)
	quitChan := make(chan bool)
	clientSub := &ClientSubscription{
		errChan: errChan,
	}
	go func() {
		for _, payload := range sds.StreamCodePayloads[blockNumber] {
			select {
			case <-quitChan:
				return
			default:
			}
			sds.PassedCodeChan <- payload
		}
		close(errChan)
	}()

	return clientSub, sds.ReturnErr
}

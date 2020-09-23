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

package sync_test

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth/mocks"
	s "github.com/vulcanize/ipld-eth-indexer/pkg/sync"
)

var _ = Describe("Service", func() {
	Describe("Sync", func() {
		It("Streams statediff.Payloads, converts them to IPLDPayloads, publishes IPLDPayloads, and indexes CIDPayloads", func() {
			wg := new(sync.WaitGroup)
			payloadChan := make(chan statediff.Payload, 1)
			quitChan := make(chan bool, 1)
			mockTransformer := &mocks.Transformer{
				ReturnErr:    nil,
				ReturnHeight: mocks.BlockNumber.Int64(),
			}
			mockStreamer := &mocks.PayloadStreamer{
				ReturnSub: &rpc.ClientSubscription{},
				StreamPayloads: []statediff.Payload{
					mocks.MockStateDiffPayload,
				},
				ReturnErr: nil,
			}
			processor := &s.Service{
				Streamer:    mockStreamer,
				Transformer: mockTransformer,
				PayloadChan: payloadChan,
				QuitChan:    quitChan,
				Workers:     1,
			}
			err := processor.Sync(wg)
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(2 * time.Second)
			close(quitChan)
			wg.Wait()
			Expect(mockTransformer.PassedStateDiff).To(Equal(mocks.MockStateDiffPayload))
			Expect(mockStreamer.PassedPayloadChan).To(Equal(payloadChan))
		})
	})
})

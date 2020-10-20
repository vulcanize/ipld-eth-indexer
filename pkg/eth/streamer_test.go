// Copyright 2019 Vulcanize
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eth_test

import (
	"github.com/ethereum/go-ethereum/statediff"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/eth/mocks"
)

var (
	mockPayload1 = statediff.Payload{
		BlockRlp: []byte{1, 2, 3, 4, 5},
	}
	mockPayload2 = statediff.Payload{
		BlockRlp: []byte{2, 3, 4, 5, 6},
	}
	mockPayload3 = statediff.Payload{
		BlockRlp: []byte{3, 4, 5, 6, 7},
	}
	mockCodeAndCodeHash1 = statediff.CodeAndCodeHash{
		Code: []byte{1, 2, 3, 4, 5},
	}
	mockCodeAndCodeHash2 = statediff.CodeAndCodeHash{
		Code: []byte{2, 3, 4, 5, 6},
	}
	mockCodeAndCodeHash3 = statediff.CodeAndCodeHash{
		Code: []byte{3, 4, 5, 6, 7},
	}
)

var _ = Describe("StateDiff Streamer", func() {
	It("subscribes to the geth statediff service", func() {
		client := &mocks.StreamClient{}
		client.StreamPayloads = []interface{}{
			mockPayload1,
			mockPayload2,
			mockPayload3,
		}
		streamer := eth.NewPayloadStreamer(client)
		payloadChan := make(chan statediff.Payload)
		sub, err := streamer.Stream(payloadChan)
		Expect(err).NotTo(HaveOccurred())
		payloads := make([]statediff.Payload, 0, 3)
	loop:
		for {
			select {
			case <-sub.Err():
				break loop
			case payload := <-payloadChan:
				payloads = append(payloads, payload)
			}
		}
		Expect(len(payloads)).To(Equal(3))
		Expect(payloads[0]).To(Equal(mockPayload1))
		Expect(payloads[1]).To(Equal(mockPayload2))
		Expect(payloads[2]).To(Equal(mockPayload3))
	})
})

var _ = Describe("CodeAndCodeHash Streamer", func() {
	It("subscribes to the geth code and codehash endpoint", func() {
		client := &mocks.StreamClient{}
		client.StreamPayloads = []interface{}{
			mockCodeAndCodeHash1,
			mockCodeAndCodeHash2,
			mockCodeAndCodeHash3,
		}
		streamer := eth.NewPayloadStreamer(client)
		payloadChan := make(chan statediff.CodeAndCodeHash)
		sub, err := streamer.StreamCodeAndCodeHash(payloadChan, 1)
		Expect(err).NotTo(HaveOccurred())
		payloads := make([]statediff.CodeAndCodeHash, 0, 3)
	loop:
		for {
			select {
			case <-sub.Err():
				break loop
			case payload := <-payloadChan:
				payloads = append(payloads, payload)
			}
		}
		Expect(len(payloads)).To(Equal(3))
		Expect(payloads[0]).To(Equal(mockCodeAndCodeHash1))
		Expect(payloads[1]).To(Equal(mockCodeAndCodeHash2))
		Expect(payloads[2]).To(Equal(mockCodeAndCodeHash3))
	})
})

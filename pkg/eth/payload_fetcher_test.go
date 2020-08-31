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

package eth_test

import (
	"time"

	"github.com/ethereum/go-ethereum/statediff"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/eth/mocks"
)

var _ = Describe("StateDiffFetcher", func() {
	Describe("FetchStateDiffsAt", func() {
		var (
			mc               *mocks.BackFillerClient
			stateDiffFetcher *eth.PayloadFetcher
			payload2         statediff.Payload
			blockNumber2     uint64
		)
		BeforeEach(func() {
			mc = new(mocks.BackFillerClient)
			err := mc.SetReturnDiffAt(mocks.BlockNumber.Uint64(), mocks.MockStateDiffPayload)
			Expect(err).ToNot(HaveOccurred())
			payload2 = mocks.MockStateDiffPayload
			payload2.BlockRlp = []byte{}
			blockNumber2 = mocks.BlockNumber.Uint64() + 1
			err = mc.SetReturnDiffAt(blockNumber2, payload2)
			Expect(err).ToNot(HaveOccurred())
			stateDiffFetcher = eth.NewPayloadFetcher(mc, time.Second*60)
		})
		It("Batch calls statediff_stateDiffAt", func() {
			blockHeights := []uint64{
				mocks.BlockNumber.Uint64(),
				blockNumber2,
			}
			stateDiffPayloads, err := stateDiffFetcher.FetchAt(blockHeights)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(stateDiffPayloads)).To(Equal(2))
			payload1 := stateDiffPayloads[0]
			payload2 := stateDiffPayloads[1]
			Expect(payload1).To(Equal(mocks.MockStateDiffPayload))
			Expect(payload2).To(Equal(payload2))
		})
	})
})

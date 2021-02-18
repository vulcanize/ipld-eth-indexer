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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/eth/mocks"
	"github.com/vulcanize/ipld-eth-indexer/pkg/postgres"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
)

var (
	mockBlock0       = newMockBlock(0)
	mockBlock1       = newMockBlock(1)
	mockBlock2       = newMockBlock(2)
	mockBlock3       = newMockBlock(3)
	mockBlock5       = newMockBlock(5)
	mockBlock1010101 = newMockBlock(1010101)
)

var _ = Describe("Retriever", func() {
	var (
		db        *postgres.DB
		repo      *eth.IPLDPublisher
		retriever *eth.GapRetriever
	)
	BeforeEach(func() {
		var err error
		db, err = shared.SetupDB()
		Expect(err).ToNot(HaveOccurred())
		repo = eth.NewIPLDPublisher(db)
		retriever = eth.NewGapRetriever(db)
	})
	AfterEach(func() {
		eth.TearDownDB(db)
	})

	Describe("RetrieveFirstBlockNumber", func() {
		It("Throws an error if there are no blocks in the database", func() {
			_, err := retriever.RetrieveFirstBlockNumber()
			Expect(err).To(HaveOccurred())
		})
		It("Gets the number of the first block that has data in the database", func() {
			err := repo.Publish(mocks.MockConvertedPayload)
			Expect(err).ToNot(HaveOccurred())
			num, err := retriever.RetrieveFirstBlockNumber()
			Expect(err).ToNot(HaveOccurred())
			Expect(num).To(Equal(int64(1)))
		})

		It("Gets the number of the first block that has data in the database", func() {
			payload := mocks.MockConvertedPayload
			payload.Block = mockBlock1010101
			err := repo.Publish(payload)
			Expect(err).ToNot(HaveOccurred())
			num, err := retriever.RetrieveFirstBlockNumber()
			Expect(err).ToNot(HaveOccurred())
			Expect(num).To(Equal(int64(1010101)))
		})

		It("Gets the number of the first block that has data in the database", func() {
			payload1 := mocks.MockConvertedPayload
			payload1.Block = mockBlock1010101
			payload2 := payload1
			payload2.Block = mockBlock5
			err := repo.Publish(payload1)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload2)
			Expect(err).ToNot(HaveOccurred())
			num, err := retriever.RetrieveFirstBlockNumber()
			Expect(err).ToNot(HaveOccurred())
			Expect(num).To(Equal(int64(5)))
		})
	})

	Describe("RetrieveLastBlockNumber", func() {
		It("Throws an error if there are no blocks in the database", func() {
			_, err := retriever.RetrieveLastBlockNumber()
			Expect(err).To(HaveOccurred())
		})
		It("Gets the number of the latest block that has data in the database", func() {
			err := repo.Publish(mocks.MockConvertedPayload)
			Expect(err).ToNot(HaveOccurred())
			num, err := retriever.RetrieveLastBlockNumber()
			Expect(err).ToNot(HaveOccurred())
			Expect(num).To(Equal(int64(1)))
		})

		It("Gets the number of the latest block that has data in the database", func() {
			payload := mocks.MockConvertedPayload
			payload.Block = mockBlock1010101
			err := repo.Publish(payload)
			Expect(err).ToNot(HaveOccurred())
			num, err := retriever.RetrieveLastBlockNumber()
			Expect(err).ToNot(HaveOccurred())
			Expect(num).To(Equal(int64(1010101)))
		})

		It("Gets the number of the latest block that has data in the database", func() {
			payload1 := mocks.MockConvertedPayload
			payload1.Block = mockBlock1010101
			payload2 := payload1
			payload2.Block = mockBlock5
			err := repo.Publish(payload1)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload2)
			Expect(err).ToNot(HaveOccurred())
			num, err := retriever.RetrieveLastBlockNumber()
			Expect(err).ToNot(HaveOccurred())
			Expect(num).To(Equal(int64(1010101)))
		})
	})

	Describe("RetrieveGapsInData", func() {
		It("Doesn't return gaps if there are none", func() {
			payload0 := mocks.MockConvertedPayload
			payload0.Block = mockBlock0
			payload1 := mocks.MockConvertedPayload
			payload2 := payload1
			payload2.Block = mockBlock2
			payload3 := payload2
			payload3.Block = mockBlock3
			err := repo.Publish(payload0)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload1)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload2)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload3)
			Expect(err).ToNot(HaveOccurred())
			gaps, err := retriever.RetrieveGapsInData(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(gaps)).To(Equal(0))
		})

		It("Returns the gap from 0 to the earliest block", func() {
			payload := mocks.MockConvertedPayload
			payload.Block = mockBlock5
			err := repo.Publish(payload)
			Expect(err).ToNot(HaveOccurred())
			gaps, err := retriever.RetrieveGapsInData(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(gaps)).To(Equal(1))
			Expect(gaps[0].Start).To(Equal(uint64(0)))
			Expect(gaps[0].Stop).To(Equal(uint64(4)))
		})

		It("Can handle single block gaps", func() {
			payload0 := mocks.MockConvertedPayload
			payload0.Block = mockBlock0
			payload1 := mocks.MockConvertedPayload
			payload3 := payload1
			payload3.Block = mockBlock3
			err := repo.Publish(payload0)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload1)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload3)
			Expect(err).ToNot(HaveOccurred())
			gaps, err := retriever.RetrieveGapsInData(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(gaps)).To(Equal(1))
			Expect(gaps[0].Start).To(Equal(uint64(2)))
			Expect(gaps[0].Stop).To(Equal(uint64(2)))
		})

		It("Finds gap between two entries", func() {
			payload1 := mocks.MockConvertedPayload
			payload1.Block = mockBlock1010101
			payload2 := payload1
			payload2.Block = mockBlock0
			err := repo.Publish(payload1)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload2)
			Expect(err).ToNot(HaveOccurred())
			gaps, err := retriever.RetrieveGapsInData(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(gaps)).To(Equal(1))
			Expect(gaps[0].Start).To(Equal(uint64(1)))
			Expect(gaps[0].Stop).To(Equal(uint64(1010100)))
		})

		It("Finds gaps between multiple entries", func() {
			payload1 := mocks.MockConvertedPayload
			payload1.Block = mockBlock1010101
			payload2 := mocks.MockConvertedPayload
			payload2.Block = mockBlock1
			payload3 := mocks.MockConvertedPayload
			payload3.Block = mockBlock5
			payload4 := mocks.MockConvertedPayload
			payload4.Block = newMockBlock(100)
			payload5 := mocks.MockConvertedPayload
			payload5.Block = newMockBlock(101)
			payload6 := mocks.MockConvertedPayload
			payload6.Block = newMockBlock(102)
			payload7 := mocks.MockConvertedPayload
			payload7.Block = newMockBlock(103)
			payload8 := mocks.MockConvertedPayload
			payload8.Block = newMockBlock(104)
			payload9 := mocks.MockConvertedPayload
			payload9.Block = newMockBlock(105)
			payload10 := mocks.MockConvertedPayload
			payload10.Block = newMockBlock(106)
			payload11 := mocks.MockConvertedPayload
			payload11.Block = newMockBlock(1000)

			err := repo.Publish(payload1)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload2)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload3)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload4)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload5)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload6)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload7)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload8)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload9)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload10)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload11)
			Expect(err).ToNot(HaveOccurred())

			gaps, err := retriever.RetrieveGapsInData(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(gaps)).To(Equal(5))
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 0, Stop: 0})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 2, Stop: 4})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 6, Stop: 99})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 107, Stop: 999})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 1001, Stop: 1010100})).To(BeTrue())
		})

		It("Finds validation level gaps", func() {

			payload1 := mocks.MockConvertedPayload
			payload1.Block = mockBlock1010101
			payload2 := mocks.MockConvertedPayload
			payload2.Block = mockBlock1
			payload3 := mocks.MockConvertedPayload
			payload3.Block = mockBlock5
			payload4 := mocks.MockConvertedPayload
			payload4.Block = newMockBlock(100)
			payload5 := mocks.MockConvertedPayload
			payload5.Block = newMockBlock(101)
			payload6 := mocks.MockConvertedPayload
			payload6.Block = newMockBlock(102)
			payload7 := mocks.MockConvertedPayload
			payload7.Block = newMockBlock(103)
			payload8 := mocks.MockConvertedPayload
			payload8.Block = newMockBlock(104)
			payload9 := mocks.MockConvertedPayload
			payload9.Block = newMockBlock(105)
			payload10 := mocks.MockConvertedPayload
			payload10.Block = newMockBlock(106)
			payload11 := mocks.MockConvertedPayload
			payload11.Block = newMockBlock(107)
			payload12 := mocks.MockConvertedPayload
			payload12.Block = newMockBlock(108)
			payload13 := mocks.MockConvertedPayload
			payload13.Block = newMockBlock(109)
			payload14 := mocks.MockConvertedPayload
			payload14.Block = newMockBlock(1000)

			err := repo.Publish(payload1)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload2)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload3)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload4)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload5)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload6)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload7)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload8)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload9)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload10)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload11)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload12)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload13)
			Expect(err).ToNot(HaveOccurred())
			err = repo.Publish(payload14)
			Expect(err).ToNot(HaveOccurred())

			cleaner := eth.NewDBCleaner(db)
			err = cleaner.ResetValidation([][2]uint64{{101, 102}, {104, 104}, {106, 108}})
			Expect(err).ToNot(HaveOccurred())

			gaps, err := retriever.RetrieveGapsInData(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(gaps)).To(Equal(8))
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 0, Stop: 0})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 2, Stop: 4})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 6, Stop: 99})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 101, Stop: 102})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 104, Stop: 104})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 106, Stop: 108})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 110, Stop: 999})).To(BeTrue())
			Expect(ListContainsGap(gaps, eth.DBGap{Start: 1001, Stop: 1010100})).To(BeTrue())
		})
	})
})

func newMockBlock(blockNumber uint64) *types.Block {
	header := mocks.MockHeader
	header.Number.SetUint64(blockNumber)
	return types.NewBlock(&mocks.MockHeader, mocks.MockTransactions, nil, mocks.MockReceipts, new(trie.Trie))
}

// ListContainsGap used to check if a list of Gaps contains a particular Gap
func ListContainsGap(gapList []eth.DBGap, gap eth.DBGap) bool {
	for _, listGap := range gapList {
		if listGap == gap {
			return true
		}
	}
	return false
}

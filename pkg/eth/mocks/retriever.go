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
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/eth"
)

// Retriever is a mock retriever for use in tests
type Retriever struct {
	GapsToRetrieve              []eth.DBGap
	GapsToRetrieveErr           error
	CalledTimes                 int
	FirstBlockNumberToReturn    int64
	RetrieveFirstBlockNumberErr error
}

// RetrieveLastBlockNumber mock method
func (*Retriever) RetrieveLastBlockNumber() (int64, error) {
	panic("implement me")
}

// RetrieveFirstBlockNumber mock method
func (mcr *Retriever) RetrieveFirstBlockNumber() (int64, error) {
	return mcr.FirstBlockNumberToReturn, mcr.RetrieveFirstBlockNumberErr
}

// RetrieveGapsInData mock method
func (mcr *Retriever) RetrieveGapsInData(int) ([]eth.DBGap, error) {
	mcr.CalledTimes++
	return mcr.GapsToRetrieve, mcr.GapsToRetrieveErr
}

// SetGapsToRetrieve mock method
func (mcr *Retriever) SetGapsToRetrieve(gaps []eth.DBGap) {
	if mcr.GapsToRetrieve == nil {
		mcr.GapsToRetrieve = make([]eth.DBGap, 0)
	}
	mcr.GapsToRetrieve = append(mcr.GapsToRetrieve, gaps...)
}

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

package eth

import (
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/postgres"
)

// Retriever interface for substituting mocks in tests
type Retriever interface {
	RetrieveFirstBlockNumber() (int64, error)
	RetrieveLastBlockNumber() (int64, error)
	RetrieveGapsInData(validationLevel int) ([]DBGap, error)
}

// GapRetriever type for Ethereum
type GapRetriever struct {
	db *postgres.DB
}

// NewRetriever returns a pointer to a new Retriever
func NewRetriever(db *postgres.DB) *GapRetriever {
	return &GapRetriever{
		db: db,
	}
}

// RetrieveFirstBlockNumber is used to retrieve the first block number in the db
func (ecr *GapRetriever) RetrieveFirstBlockNumber() (int64, error) {
	var blockNumber int64
	err := ecr.db.Get(&blockNumber, "SELECT block_number FROM eth.header_cids ORDER BY block_number ASC LIMIT 1")
	return blockNumber, err
}

// RetrieveLastBlockNumber is used to retrieve the latest block number in the db
func (ecr *GapRetriever) RetrieveLastBlockNumber() (int64, error) {
	var blockNumber int64
	err := ecr.db.Get(&blockNumber, "SELECT block_number FROM eth.header_cids ORDER BY block_number DESC LIMIT 1 ")
	return blockNumber, err
}

type DBGap struct {
	Start uint64 `db:"start"`
	Stop  uint64 `db:"stop"`
}

// RetrieveGapsInData is used to find the the block numbers at which we are missing data in the db
// it finds the union of heights where no data exists and where the times_validated is lower than the validation level
func (ecr *GapRetriever) RetrieveGapsInData(validationLevel int) ([]DBGap, error) {
	log.Info("searching for gaps in the eth ipfs watcher database")
	startingBlock, err := ecr.RetrieveFirstBlockNumber()
	if err != nil {
		return nil, fmt.Errorf("eth CIDRetriever RetrieveFirstBlockNumber error: %v", err)
	}
	var initialGap []DBGap
	if startingBlock != 0 {
		stop := uint64(startingBlock - 1)
		log.Infof("found gap at the beginning of the eth sync from 0 to %d", stop)
		initialGap = []DBGap{{
			Start: 0,
			Stop:  stop,
		}}
	}

	pgStr := `SELECT header_cids.block_number + 1 AS start, min(fr.block_number) - 1 AS stop FROM eth.header_cids
				LEFT JOIN eth.header_cids r on eth.header_cids.block_number = r.block_number - 1
				LEFT JOIN eth.header_cids fr on eth.header_cids.block_number < fr.block_number
				WHERE r.block_number is NULL and fr.block_number IS NOT NULL
				GROUP BY header_cids.block_number, r.block_number`
	emptyGaps := make([]DBGap, 0)
	if err := ecr.db.Select(&emptyGaps, pgStr); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Find sections of blocks where we are below the validation level
	// There will be no overlap between these "gaps" and the ones above
	pgStr = `SELECT block_number FROM eth.header_cids
			WHERE times_validated < $1
			ORDER BY block_number`
	var heights []uint64
	if err := ecr.db.Select(&heights, pgStr, validationLevel); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return append(append(initialGap, emptyGaps...), MissingHeightsToGaps(heights)...), nil
}

// MissingHeightsToGaps returns a slice of gaps from a slice of missing block heights
func MissingHeightsToGaps(heights []uint64) []DBGap {
	if len(heights) == 0 {
		return nil
	}
	validationGaps := make([]DBGap, 0)
	start := heights[0]
	lastHeight := start
	for i, height := range heights[1:] {
		if height != lastHeight+1 {
			validationGaps = append(validationGaps, DBGap{
				Start: start,
				Stop:  lastHeight,
			})
			start = height
		}
		if i+2 == len(heights) {
			validationGaps = append(validationGaps, DBGap{
				Start: start,
				Stop:  height,
			})
		}
		lastHeight = height
	}
	return validationGaps
}

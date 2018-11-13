// Copyright 2018 Vulcanize
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

package stability_fee

import (
	"fmt"
	"github.com/vulcanize/vulcanizedb/pkg/core"
	"github.com/vulcanize/vulcanizedb/pkg/datastore/postgres"
	"github.com/vulcanize/vulcanizedb/pkg/transformers/shared"
	"github.com/vulcanize/vulcanizedb/pkg/transformers/shared/constants"
)

type PitFileStabilityFeeRepository struct {
	db *postgres.DB
}

func (repository PitFileStabilityFeeRepository) Create(headerID int64, models []interface{}) error {
	tx, err := repository.db.Begin()
	if err != nil {
		return err
	}

	for _, model := range models {
		pitFileSF, ok := model.(PitFileStabilityFeeModel)
		if !ok {
			tx.Rollback()
			return fmt.Errorf("model of type %T, not %T", model, PitFileStabilityFeeModel{})
		}

		_, err = tx.Exec(
			`INSERT into maker.pit_file_stability_fee (header_id, what, data, log_idx, tx_idx, raw_log)
        VALUES($1, $2, $3, $4, $5, $6)`,
			headerID, pitFileSF.What, pitFileSF.Data, pitFileSF.LogIndex, pitFileSF.TransactionIndex, pitFileSF.Raw,
		)

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = shared.MarkHeaderCheckedInTransaction(headerID, tx, constants.PitFileStabilityFeeChecked)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (repository PitFileStabilityFeeRepository) MarkHeaderChecked(headerID int64) error {
	return shared.MarkHeaderChecked(headerID, repository.db, constants.PitFileStabilityFeeChecked)
}

func (repository PitFileStabilityFeeRepository) MissingHeaders(startingBlockNumber, endingBlockNumber int64) ([]core.Header, error) {
	return shared.MissingHeaders(startingBlockNumber, endingBlockNumber, repository.db, constants.PitFileStabilityFeeChecked)
}

func (repository *PitFileStabilityFeeRepository) SetDB(db *postgres.DB) {
	repository.db = db
}
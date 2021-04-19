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
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"github.com/vulcanize/ipld-eth-indexer/pkg/postgres"
	"github.com/vulcanize/ipld-eth-indexer/pkg/prom"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
)

var (
	nullHash = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
)

// Indexer interface to allow substituting mocks in tests
type Indexer interface {
	Index(cids CIDPayload) error
}

// Indexer satisfies the Indexer interface for ethereum
type CIDIndexer struct {
	db *postgres.DB
}

// NewCIDIndexer creates a new pointer to a Indexer which satisfies the CIDIndexer interface
func NewCIDIndexer(db *postgres.DB) *CIDIndexer {
	return &CIDIndexer{
		db: db,
	}
}

// Index indexes a cidPayload in Postgres
func (in *CIDIndexer) Index(cids CIDPayload) error {
	// Begin new db tx
	tx, err := in.db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			shared.Rollback(tx)
			panic(p)
		} else if err != nil {
			shared.Rollback(tx)
		} else {
			err = tx.Commit()
		}
	}()

	var headerID int64
	headerID, err = in.indexHeaderCID(tx, cids.HeaderCID)
	if err != nil {
		log.Error("eth indexer error when indexing header")
		return err
	}
	for _, uncle := range cids.UncleCIDs {
		err = in.indexUncleCID(tx, uncle, headerID)
		if err != nil {
			log.Error("eth indexer error when indexing uncle")
			return err
		}
	}
	err = in.indexTransactionAndReceiptCIDs(tx, cids, headerID)
	if err != nil {
		log.Error("eth indexer error when indexing transactions and receipts")
		return err
	}
	err = in.indexStateAndStorageCIDs(tx, cids, headerID)
	if err != nil {
		log.Error("eth indexer error when indexing state and storage nodes")
	}
	return err
}

func (in *CIDIndexer) indexHeaderCID(tx *sqlx.Tx, header HeaderModel) (int64, error) {
	var headerID int64
	err := tx.QueryRowx(`INSERT INTO eth.header_cids (block_number, block_hash, parent_hash, cid, td, node_id, reward, state_root, tx_root, receipt_root, uncle_root, bloom, timestamp, mh_key, times_validated)
								VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
								ON CONFLICT (block_number, block_hash) DO UPDATE SET (parent_hash, cid, td, node_id, reward, state_root, tx_root, receipt_root, uncle_root, bloom, timestamp, mh_key, times_validated) = ($3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, eth.header_cids.times_validated + 1)
								RETURNING id`,
		header.BlockNumber, header.BlockHash, header.ParentHash, header.CID, header.TotalDifficulty, in.db.NodeID, header.Reward, header.StateRoot, header.TxRoot,
		header.RctRoot, header.UncleRoot, header.Bloom, header.Timestamp, header.MhKey, 1).Scan(&headerID)
	if err == nil {
		prom.BlockInc()
	}
	return headerID, err
}

func (in *CIDIndexer) indexUncleCID(tx *sqlx.Tx, uncle UncleModel, headerID int64) error {
	_, err := tx.Exec(`INSERT INTO eth.uncle_cids (block_hash, header_id, parent_hash, cid, reward, mh_key) VALUES ($1, $2, $3, $4, $5, $6)
								ON CONFLICT (header_id, block_hash) DO UPDATE SET (parent_hash, cid, reward, mh_key) = ($3, $4, $5, $6)`,
		uncle.BlockHash, headerID, uncle.ParentHash, uncle.CID, uncle.Reward, uncle.MhKey)
	return err
}

func (in *CIDIndexer) indexTransactionAndReceiptCIDs(tx *sqlx.Tx, payload CIDPayload, headerID int64) error {
	for _, trxCidMeta := range payload.TransactionCIDs {
		var txID int64
		err := tx.QueryRowx(`INSERT INTO eth.transaction_cids (header_id, tx_hash, cid, dst, src, index, mh_key, tx_data) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
									ON CONFLICT (header_id, tx_hash) DO UPDATE SET (cid, dst, src, index, mh_key, tx_data) = ($3, $4, $5, $6, $7, $8)
									RETURNING id`,
			headerID, trxCidMeta.TxHash, trxCidMeta.CID, trxCidMeta.Dst, trxCidMeta.Src, trxCidMeta.Index, trxCidMeta.MhKey, trxCidMeta.Data).Scan(&txID)
		if err != nil {
			return err
		}
		prom.TransactionInc()
		receiptCidMeta, ok := payload.ReceiptCIDs[common.HexToHash(trxCidMeta.TxHash)]
		if ok {
			if err := in.indexReceiptCID(tx, receiptCidMeta, txID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (in *CIDIndexer) indexTransactionCID(tx *sqlx.Tx, transaction TxModel, headerID int64) (int64, error) {
	var txID int64
	err := tx.QueryRowx(`INSERT INTO eth.transaction_cids (header_id, tx_hash, cid, dst, src, index, mh_key, tx_data) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
									ON CONFLICT (header_id, tx_hash) DO UPDATE SET (cid, dst, src, index, mh_key, tx_data) = ($3, $4, $5, $6, $7, $8)
									RETURNING id`,
		headerID, transaction.TxHash, transaction.CID, transaction.Dst, transaction.Src, transaction.Index, transaction.MhKey, transaction.Data).Scan(&txID)
	if err == nil {
		prom.TransactionInc()
	}
	return txID, err
}

func (in *CIDIndexer) indexReceiptCID(tx *sqlx.Tx, rct ReceiptModel, txID int64) error {
	_, err := tx.Exec(`INSERT INTO eth.receipt_cids (tx_id, cid, contract, contract_hash, topic0s, topic1s, topic2s, topic3s, log_contracts, mh_key, post_state, post_status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
							  ON CONFLICT (tx_id) DO UPDATE SET (cid, contract, contract_hash, topic0s, topic1s, topic2s, topic3s, log_contracts, mh_key, post_state, post_status) = ($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		txID, rct.CID, rct.Contract, rct.ContractHash, rct.Topic0s, rct.Topic1s, rct.Topic2s, rct.Topic3s, rct.LogContracts, rct.MhKey, rct.PostState, rct.PostStatus)
	if err == nil {
		prom.ReceiptInc()
	}
	return err
}

func (in *CIDIndexer) indexStateAndStorageCIDs(tx *sqlx.Tx, payload CIDPayload, headerID int64) error {
	for _, stateCID := range payload.StateNodeCIDs {
		var stateID int64
		var stateKey string
		if stateCID.StateKey != nullHash.String() {
			stateKey = stateCID.StateKey
		}
		err := tx.QueryRowx(`INSERT INTO eth.state_cids (header_id, state_leaf_key, cid, state_path, node_type, diff, mh_key) VALUES ($1, $2, $3, $4, $5, $6, $7)
									ON CONFLICT (header_id, state_path) DO UPDATE SET (state_leaf_key, cid, node_type, diff, mh_key) = ($2, $3, $5, $6, $7)
									RETURNING id`,
			headerID, stateKey, stateCID.CID, stateCID.Path, stateCID.NodeType, true, stateCID.MhKey).Scan(&stateID)
		if err != nil {
			return err
		}
		// If we have a state leaf node, index the associated account and storage nodes
		if stateCID.NodeType == 2 {
			statePath := common.Bytes2Hex(stateCID.Path)
			for _, storageCID := range payload.StorageNodeCIDs[statePath] {
				if err := in.indexStorageCID(tx, storageCID, stateID); err != nil {
					return err
				}
			}
			if stateAccount, ok := payload.StateAccounts[statePath]; ok {
				if err := in.indexStateAccount(tx, stateAccount, stateID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (in *CIDIndexer) indexStateCID(tx *sqlx.Tx, stateNode StateNodeModel, headerID int64) (int64, error) {
	var stateID int64
	var stateKey string
	if stateNode.StateKey != nullHash.String() {
		stateKey = stateNode.StateKey
	}
	err := tx.QueryRowx(`INSERT INTO eth.state_cids (header_id, state_leaf_key, cid, state_path, node_type, diff, mh_key) VALUES ($1, $2, $3, $4, $5, $6, $7)
									ON CONFLICT (header_id, state_path) DO UPDATE SET (state_leaf_key, cid, node_type, diff, mh_key) = ($2, $3, $5, $6, $7)
									RETURNING id`,
		headerID, stateKey, stateNode.CID, stateNode.Path, stateNode.NodeType, true, stateNode.MhKey).Scan(&stateID)
	return stateID, err
}

func (in *CIDIndexer) indexStateAccount(tx *sqlx.Tx, stateAccount StateAccountModel, stateID int64) error {
	_, err := tx.Exec(`INSERT INTO eth.state_accounts (state_id, balance, nonce, code_hash, storage_root) VALUES ($1, $2, $3, $4, $5)
							  ON CONFLICT (state_id) DO UPDATE SET (balance, nonce, code_hash, storage_root) = ($2, $3, $4, $5)`,
		stateID, stateAccount.Balance, stateAccount.Nonce, stateAccount.CodeHash, stateAccount.StorageRoot)
	return err
}

func (in *CIDIndexer) indexStorageCID(tx *sqlx.Tx, storageCID StorageNodeModel, stateID int64) error {
	var storageKey string
	if storageCID.StorageKey != nullHash.String() {
		storageKey = storageCID.StorageKey
	}
	_, err := tx.Exec(`INSERT INTO eth.storage_cids (state_id, storage_leaf_key, cid, storage_path, node_type, diff, mh_key) VALUES ($1, $2, $3, $4, $5, $6, $7) 
							  ON CONFLICT (state_id, storage_path) DO UPDATE SET (storage_leaf_key, cid, node_type, diff, mh_key) = ($2, $3, $5, $6, $7)`,
		stateID, storageKey, storageCID.CID, storageCID.Path, storageCID.NodeType, true, storageCID.MhKey)
	return err
}

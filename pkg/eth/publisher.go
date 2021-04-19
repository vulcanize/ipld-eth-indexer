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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
	sdtypes "github.com/ethereum/go-ethereum/statediff/types"
	"github.com/jmoiron/sqlx"
	"github.com/multiformats/go-multihash"

	"github.com/vulcanize/ipld-eth-indexer/pkg/ipfs/ipld"
	"github.com/vulcanize/ipld-eth-indexer/pkg/postgres"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
)

// Publisher interface for substituting mocks in tests
type Publisher interface {
	Publish(payload ConvertedPayload) error
}

// IPLDPublisher satisfies the IPLDPublisher interface for ethereum
// It interfaces directly with the public.blocks table of PG-IPFS rather than going through an ipfs intermediary
// It publishes and indexes IPLDs together in a single sqlx.Tx
type IPLDPublisher struct {
	indexer *CIDIndexer
}

// NewIPLDPublisher creates a pointer to a new IPLDPublisher which satisfies the IPLDPublisher interface
func NewIPLDPublisher(db *postgres.DB) *IPLDPublisher {
	return &IPLDPublisher{
		indexer: NewCIDIndexer(db),
	}
}

// Publish publishes an IPLDPayload to IPFS and returns the corresponding CIDPayload
func (pub *IPLDPublisher) Publish(payload ConvertedPayload) error {
	// Generate the iplds
	headerNode, uncleNodes, txNodes, txTrieNodes, rctNodes, rctTrieNodes, err := ipld.FromBlockAndReceipts(payload.Block, payload.Receipts)
	if err != nil {
		return err
	}

	// Begin new db tx
	tx, err := pub.indexer.db.Beginx()
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

	// Publish trie nodes
	for _, node := range txTrieNodes {
		err = shared.PublishIPLD(tx, node)
		if err != nil {
			return err
		}
	}
	for _, node := range rctTrieNodes {
		err = shared.PublishIPLD(tx, node)
		if err != nil {
			return err
		}
	}

	// Publish and index header
	err = shared.PublishIPLD(tx, headerNode)
	if err != nil {
		return err
	}
	reward := CalcEthBlockReward(payload.Block.Header(), payload.Block.Uncles(), payload.Block.Transactions(), payload.Receipts)
	header := HeaderModel{
		CID:             headerNode.Cid().String(),
		MhKey:           shared.MultihashKeyFromCID(headerNode.Cid()),
		ParentHash:      payload.Block.ParentHash().String(),
		BlockNumber:     payload.Block.Number().String(),
		BlockHash:       payload.Block.Hash().String(),
		TotalDifficulty: payload.TotalDifficulty.String(),
		Reward:          reward.String(),
		Bloom:           payload.Block.Bloom().Bytes(),
		StateRoot:       payload.Block.Root().String(),
		RctRoot:         payload.Block.ReceiptHash().String(),
		TxRoot:          payload.Block.TxHash().String(),
		UncleRoot:       payload.Block.UncleHash().String(),
		Timestamp:       payload.Block.Time(),
	}
	var headerID int64
	headerID, err = pub.indexer.indexHeaderCID(tx, header)
	if err != nil {
		return err
	}

	// Publish and index uncles
	for _, uncleNode := range uncleNodes {
		err = shared.PublishIPLD(tx, uncleNode)
		if err != nil {
			return err
		}
		uncleReward := CalcUncleMinerReward(payload.Block.Number().Uint64(), uncleNode.Number.Uint64())
		uncle := UncleModel{
			CID:        uncleNode.Cid().String(),
			MhKey:      shared.MultihashKeyFromCID(uncleNode.Cid()),
			ParentHash: uncleNode.ParentHash.String(),
			BlockHash:  uncleNode.Hash().String(),
			Reward:     uncleReward.String(),
		}
		err = pub.indexer.indexUncleCID(tx, uncle, headerID)
		if err != nil {
			return err
		}
	}

	// Publish and index txs and receipts
	for i, txNode := range txNodes {
		err = shared.PublishIPLD(tx, txNode)
		if err != nil {
			return err
		}
		rctNode := rctNodes[i]
		err = shared.PublishIPLD(tx, rctNode)
		if err != nil {
			return err
		}
		txModel := payload.TxMetaData[i]
		txModel.CID = txNode.Cid().String()
		txModel.MhKey = shared.MultihashKeyFromCID(txNode.Cid())
		var txID int64
		txID, err = pub.indexer.indexTransactionCID(tx, txModel, headerID)
		if err != nil {
			return err
		}
		rctModel := payload.ReceiptMetaData[i]
		rctModel.CID = rctNode.Cid().String()
		rctModel.MhKey = shared.MultihashKeyFromCID(rctNode.Cid())
		if len(payload.Receipts[i].PostState) == 0 {
			rctModel.PostStatus = payload.Receipts[i].Status
		} else {
			rctModel.PostState = common.Bytes2Hex(payload.Receipts[i].PostState)
		}
		err = pub.indexer.indexReceiptCID(tx, rctModel, txID)
		if err != nil {
			return err
		}
	}

	// Publish and index state and storage
	err = pub.publishAndIndexStateAndStorage(tx, payload, headerID)

	return err // return err variable explicitly so that we return the err = tx.Commit() assignment in the defer
}

func (pub *IPLDPublisher) publishAndIndexStateAndStorage(tx *sqlx.Tx, payload ConvertedPayload, headerID int64) error {
	// Publish and index state and storage
	for _, stateNode := range payload.StateNodes {
		stateCIDStr, err := shared.PublishRaw(tx, ipld.MEthStateTrie, multihash.KECCAK_256, stateNode.Value)
		if err != nil {
			return err
		}
		mhKey, _ := shared.MultihashKeyFromCIDString(stateCIDStr)
		stateModel := StateNodeModel{
			Path:     stateNode.Path,
			StateKey: stateNode.LeafKey.String(),
			CID:      stateCIDStr,
			MhKey:    mhKey,
			NodeType: ResolveFromNodeType(stateNode.Type),
		}
		stateID, err := pub.indexer.indexStateCID(tx, stateModel, headerID)
		if err != nil {
			return err
		}
		// If we have a leaf, decode and index the account data and any associated storage diffs
		if stateNode.Type == sdtypes.Leaf {
			var i []interface{}
			if err := rlp.DecodeBytes(stateNode.Value, &i); err != nil {
				return err
			}
			if len(i) != 2 {
				return fmt.Errorf("eth IPLDPublisher expected state leaf node rlp to decode into two elements")
			}
			var account state.Account
			if err := rlp.DecodeBytes(i[1].([]byte), &account); err != nil {
				return err
			}
			accountModel := StateAccountModel{
				Balance:     account.Balance.String(),
				Nonce:       account.Nonce,
				CodeHash:    account.CodeHash,
				StorageRoot: account.Root.String(),
			}
			if err := pub.indexer.indexStateAccount(tx, accountModel, stateID); err != nil {
				return err
			}
			for _, storageNode := range payload.StorageNodes[common.Bytes2Hex(stateNode.Path)] {
				storageCIDStr, err := shared.PublishRaw(tx, ipld.MEthStorageTrie, multihash.KECCAK_256, storageNode.Value)
				if err != nil {
					return err
				}
				mhKey, _ := shared.MultihashKeyFromCIDString(storageCIDStr)
				storageModel := StorageNodeModel{
					Path:       storageNode.Path,
					StorageKey: storageNode.LeafKey.Hex(),
					CID:        storageCIDStr,
					MhKey:      mhKey,
					NodeType:   ResolveFromNodeType(storageNode.Type),
				}
				if err := pub.indexer.indexStorageCID(tx, storageModel, stateID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

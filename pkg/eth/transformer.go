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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"
	sdtypes "github.com/ethereum/go-ethereum/statediff/types"
	node "github.com/ipfs/go-ipld-format"
	"github.com/jmoiron/sqlx"
	"github.com/multiformats/go-multihash"
	"github.com/sirupsen/logrus"

	"github.com/vulcanize/ipld-eth-indexer/pkg/ipfs/ipld"
	"github.com/vulcanize/ipld-eth-indexer/pkg/postgres"
	"github.com/vulcanize/ipld-eth-indexer/pkg/prom"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
)

// Transformer interface to allow substitution of mocks for testing
type Transformer interface {
	Transform(workerID int, payload statediff.Payload) (uint64, error)
}

// StateDiffTransformer satisfies the Transformer interface for ethereum statediff objects
type StateDiffTransformer struct {
	chainConfig *params.ChainConfig
	indexer     *CIDIndexer
}

// NewStateDiffTransformer creates a pointer to a new PayloadConverter which satisfies the PayloadConverter interface
func NewStateDiffTransformer(chainConfig *params.ChainConfig, db *postgres.DB) *StateDiffTransformer {
	return &StateDiffTransformer{
		chainConfig: chainConfig,
		indexer:     NewCIDIndexer(db),
	}
}

// Transform method is used to process statediff.Payload objects
// It performs the necessary data conversions and database persistence
func (sdt *StateDiffTransformer) Transform(workerID int, payload statediff.Payload) (uint64, error) {
	start, t := time.Now(), time.Now()
	// Unpack block rlp to access fields
	block := new(types.Block)
	if err := rlp.DecodeBytes(payload.BlockRlp, block); err != nil {
		return 0, fmt.Errorf("error decoding payload block rlp: %s", err.Error())
	}
	blockHash := block.Hash()
	blockHashStr := blockHash.String()
	height := block.NumberU64()
	traceMsg := fmt.Sprintf("worker %d transformer stats for payload at %d with hash %s:\r\n", workerID, height, blockHashStr)
	transactions := block.Transactions()
	// Decode receipts for this block
	receipts := make(types.Receipts, 0)
	if err := rlp.DecodeBytes(payload.ReceiptsRlp, &receipts); err != nil {
		return 0, fmt.Errorf("error decoding payload receipts rlp: %s", err.Error())
	}
	// Decode state diff rlp for this block
	stateDiff := new(statediff.StateObject)
	if err := rlp.DecodeBytes(payload.StateObjectRlp, stateDiff); err != nil {
		return 0, fmt.Errorf("error decoding payload state object rlp: %s", err.Error())
	}
	// Derive any missing fields
	if err := receipts.DeriveFields(sdt.chainConfig, blockHash, height, transactions); err != nil {
		return 0, err
	}
	// Generate the block iplds
	headerNode, uncleNodes, txNodes, txTrieNodes, rctNodes, rctTrieNodes, err := ipld.FromBlockAndReceipts(block, receipts)
	if err != nil {
		return 0, err
	}
	if len(txNodes) != len(txTrieNodes) && len(rctNodes) != len(rctTrieNodes) && len(txNodes) != len(rctNodes) {
		return 0, fmt.Errorf("expected number of transactions (%d), transaction trie nodes (%d), receipts (%d), and receipt trie nodes (%d)to be equal", len(txNodes), len(txTrieNodes), len(rctNodes), len(rctTrieNodes))
	}
	// Calculate reward
	reward := CalcEthBlockReward(block.Header(), block.Uncles(), block.Transactions(), receipts)
	tDiff := time.Now().Sub(t)
	prom.SetTimeMetric("t_payload_decode", tDiff)
	traceMsg += fmt.Sprintf("payload decoding time: %s\r\n", tDiff.String())
	t = time.Now()
	// Begin new db tx for everything
	tx, err := sdt.indexer.db.Beginx()
	if err != nil {
		return 0, err
	}
	// defer to handle transaction commit or rollback for any return case
	defer func() {
		if p := recover(); p != nil {
			shared.Rollback(tx)
			panic(p)
		} else if err != nil {
			shared.Rollback(tx)
		} else {
			err = tx.Commit()
			tDiff := time.Now().Sub(t)
			prom.SetTimeMetric("t_postgres_commit", tDiff)
			traceMsg += fmt.Sprintf("postgres transaction commit duration: %s\r\n", tDiff.String())
		}
		traceMsg += fmt.Sprintf(" TOTAL PROCESSING TIME: %s\r\n", time.Now().Sub(start).String())
		logrus.Trace(traceMsg)
	}()
	tDiff = time.Now().Sub(t)
	prom.SetTimeMetric("t_free_postgres", tDiff)
	traceMsg += fmt.Sprintf("time spent waiting for free postgres tx: %s:\r\n", tDiff.String())
	t = time.Now()

	// Publish and index header, collect headerID
	headerID, err := sdt.processHeader(tx, block.Header(), headerNode, reward, payload.TotalDifficulty)
	if err != nil {
		return 0, err
	}
	tDiff = time.Now().Sub(t)
	prom.SetTimeMetric("t_header_processing", tDiff)
	traceMsg += fmt.Sprintf("header processing time: %s\r\n", tDiff.String())
	t = time.Now()
	// Publish and index uncles
	if err := sdt.processUncles(tx, headerID, height, uncleNodes); err != nil {
		return 0, err
	}
	tDiff = time.Now().Sub(t)
	prom.SetTimeMetric("t_uncle_processing", tDiff)
	traceMsg += fmt.Sprintf("uncle processing time: %s\r\n", tDiff.String())
	t = time.Now()
	// Publish and index receipts and txs
	if err := sdt.processReceiptsAndTxs(tx, processArgs{
		headerID:     headerID,
		blockNumber:  block.Number(),
		receipts:     receipts,
		txs:          transactions,
		rctNodes:     rctNodes,
		rctTrieNodes: rctTrieNodes,
		txNodes:      txNodes,
		txTrieNodes:  txTrieNodes,
	}); err != nil {
		return 0, err
	}
	tDiff = time.Now().Sub(t)
	prom.SetTimeMetric("t_tx_receipt_processing", tDiff)
	traceMsg += fmt.Sprintf("tx and receipt processing time: %s\r\n", tDiff.String())
	t = time.Now()
	// Publish and index state and storage nodes
	if err := sdt.processStateAndStorage(tx, headerID, stateDiff); err != nil {
		return 0, err
	}
	tDiff = time.Now().Sub(t)
	prom.SetTimeMetric("t_state_store_processing", tDiff)
	traceMsg += fmt.Sprintf("state and storage processing time: %s\r\n", tDiff.String())
	t = time.Now()
	if err := sdt.processCodeAndCodeHashes(tx, stateDiff.CodeAndCodeHashes); err != nil {
		return 0, err
	}
	tDiff = time.Now().Sub(t)
	prom.SetTimeMetric("t_code_codehash_processing", tDiff)
	traceMsg += fmt.Sprintf("code and codehash processing time: %s\r\n", tDiff.String())
	t = time.Now()
	return height, err // return error explicity so that the defer() assigns to it
}

// processHeader publishes and indexes a header IPLD in Postgres
// it returns the headerID
func (sdt *StateDiffTransformer) processHeader(tx *sqlx.Tx, header *types.Header, headerNode node.Node, reward, td *big.Int) (int64, error) {
	// publish header
	if err := shared.PublishIPLD(tx, headerNode); err != nil {
		return 0, err
	}
	// index header
	return sdt.indexer.indexHeaderCID(tx, HeaderModel{
		CID:             headerNode.Cid().String(),
		MhKey:           shared.MultihashKeyFromCID(headerNode.Cid()),
		ParentHash:      header.ParentHash.String(),
		BlockNumber:     header.Number.String(),
		BlockHash:       header.Hash().String(),
		TotalDifficulty: td.String(),
		Reward:          reward.String(),
		Bloom:           header.Bloom.Bytes(),
		StateRoot:       header.Root.String(),
		RctRoot:         header.ReceiptHash.String(),
		TxRoot:          header.TxHash.String(),
		UncleRoot:       header.UncleHash.String(),
		Timestamp:       header.Time,
	})
}

func (sdt *StateDiffTransformer) processUncles(tx *sqlx.Tx, headerID int64, blockNumber uint64, uncleNodes []*ipld.EthHeader) error {
	// publish and index uncles
	for _, uncleNode := range uncleNodes {
		if err := shared.PublishIPLD(tx, uncleNode); err != nil {
			return err
		}
		uncleReward := CalcUncleMinerReward(blockNumber, uncleNode.Number.Uint64())
		uncle := UncleModel{
			CID:        uncleNode.Cid().String(),
			MhKey:      shared.MultihashKeyFromCID(uncleNode.Cid()),
			ParentHash: uncleNode.ParentHash.String(),
			BlockHash:  uncleNode.Hash().String(),
			Reward:     uncleReward.String(),
		}
		if err := sdt.indexer.indexUncleCID(tx, uncle, headerID); err != nil {
			return err
		}
	}
	return nil
}

// processArgs bundles arugments to processReceiptsAndTxs
type processArgs struct {
	headerID     int64
	blockNumber  *big.Int
	receipts     types.Receipts
	txs          types.Transactions
	rctNodes     []*ipld.EthReceipt
	rctTrieNodes []*ipld.EthRctTrie
	txNodes      []*ipld.EthTx
	txTrieNodes  []*ipld.EthTxTrie
}

// processReceiptsAndTxs publishes and indexes receipt and transaction IPLDs in Postgres
func (sdt *StateDiffTransformer) processReceiptsAndTxs(tx *sqlx.Tx, args processArgs) error {
	// Process receipts and txs
	signer := types.MakeSigner(sdt.chainConfig, args.blockNumber)
	for i, receipt := range args.receipts {
		// tx that corresponds with this receipt
		trx := args.txs[i]
		from, err := types.Sender(signer, trx)
		if err != nil {
			return err
		}

		// Publishing
		// publish trie nodes, these aren't indexed directly
		if err := shared.PublishIPLD(tx, args.txTrieNodes[i]); err != nil {
			return err
		}
		if err := shared.PublishIPLD(tx, args.rctTrieNodes[i]); err != nil {
			return err
		}
		// publish the txs and receipts
		txNode, rctNode := args.txNodes[i], args.rctNodes[i]
		if err := shared.PublishIPLD(tx, txNode); err != nil {
			return err
		}
		if err := shared.PublishIPLD(tx, rctNode); err != nil {
			return err
		}

		// Indexing
		// extract topic and contract data from the receipt for indexing
		topicSets := make([][]string, 4)
		mappedContracts := make(map[string]bool) // use map to avoid duplicate addresses
		for _, log := range receipt.Logs {
			for i, topic := range log.Topics {
				topicSets[i] = append(topicSets[i], topic.Hex())
			}
			mappedContracts[log.Address.String()] = true
		}
		// these are the contracts seen in the logs
		logContracts := make([]string, 0, len(mappedContracts))
		for addr := range mappedContracts {
			logContracts = append(logContracts, addr)
		}
		// this is the contract address if this receipt is for a contract creation tx
		contract := shared.HandleZeroAddr(receipt.ContractAddress)
		var contractHash string
		if contract != "" {
			contractHash = crypto.Keccak256Hash(common.HexToAddress(contract).Bytes()).String()
		}
		// index tx first so that the receipt can reference it by FK
		txModel := TxModel{
			Dst:    shared.HandleZeroAddrPointer(trx.To()),
			Src:    shared.HandleZeroAddr(from),
			TxHash: trx.Hash().String(),
			Index:  int64(i),
			Data:   trx.Data(),
			CID:    txNode.Cid().String(),
			MhKey:  shared.MultihashKeyFromCID(txNode.Cid()),
		}
		txID, err := sdt.indexer.indexTransactionCID(tx, txModel, args.headerID)
		if err != nil {
			return err
		}
		// index the receipt
		rctModel := ReceiptModel{
			Topic0s:      topicSets[0],
			Topic1s:      topicSets[1],
			Topic2s:      topicSets[2],
			Topic3s:      topicSets[3],
			Contract:     contract,
			ContractHash: contractHash,
			LogContracts: logContracts,
			CID:          rctNode.Cid().String(),
			MhKey:        shared.MultihashKeyFromCID(rctNode.Cid()),
		}
		if len(receipt.PostState) == 0 {
			rctModel.PostStatus = receipt.Status
		} else {
			rctModel.PostState = common.Bytes2Hex(receipt.PostState)
		}
		if err := sdt.indexer.indexReceiptCID(tx, rctModel, txID); err != nil {
			return err
		}
	}
	return nil
}

// processStateAndStorage publishes and indexes state and storage nodes in Postgres
func (sdt *StateDiffTransformer) processStateAndStorage(tx *sqlx.Tx, headerID int64, stateDiff *statediff.StateObject) error {
	for _, stateNode := range stateDiff.Nodes {
		// publish the state node
		stateCIDStr, err := shared.PublishRaw(tx, ipld.MEthStateTrie, multihash.KECCAK_256, stateNode.NodeValue)
		if err != nil {
			return err
		}
		mhKey, _ := shared.MultihashKeyFromCIDString(stateCIDStr)
		stateModel := StateNodeModel{
			Path:     stateNode.Path,
			StateKey: common.BytesToHash(stateNode.LeafKey).String(),
			CID:      stateCIDStr,
			MhKey:    mhKey,
			NodeType: ResolveFromNodeType(stateNode.NodeType),
		}
		// index the state node, collect the stateID to reference by FK
		stateID, err := sdt.indexer.indexStateCID(tx, stateModel, headerID)
		if err != nil {
			return err
		}
		// if we have a leaf, decode and index the account data
		if stateNode.NodeType == sdtypes.Leaf {
			var i []interface{}
			if err := rlp.DecodeBytes(stateNode.NodeValue, &i); err != nil {
				return fmt.Errorf("error decoding state leaf node rlp: %s", err.Error())
			}
			if len(i) != 2 {
				return fmt.Errorf("eth IPLDPublisher expected state leaf node rlp to decode into two elements")
			}
			var account state.Account
			if err := rlp.DecodeBytes(i[1].([]byte), &account); err != nil {
				return fmt.Errorf("error decoding state account rlp: %s", err.Error())
			}
			accountModel := StateAccountModel{
				Balance:     account.Balance.String(),
				Nonce:       account.Nonce,
				CodeHash:    account.CodeHash,
				StorageRoot: account.Root.String(),
			}
			if err := sdt.indexer.indexStateAccount(tx, accountModel, stateID); err != nil {
				return err
			}
		}
		// if there are any storage nodes associated with this node, publish and index them
		for _, storageNode := range stateNode.StorageNodes {
			storageCIDStr, err := shared.PublishRaw(tx, ipld.MEthStorageTrie, multihash.KECCAK_256, storageNode.NodeValue)
			if err != nil {
				return err
			}
			mhKey, _ := shared.MultihashKeyFromCIDString(storageCIDStr)
			storageModel := StorageNodeModel{
				Path:       storageNode.Path,
				StorageKey: common.BytesToHash(storageNode.LeafKey).String(),
				CID:        storageCIDStr,
				MhKey:      mhKey,
				NodeType:   ResolveFromNodeType(storageNode.NodeType),
			}
			if err := sdt.indexer.indexStorageCID(tx, storageModel, stateID); err != nil {
				return err
			}
		}
	}
	return nil
}

// processCodeAndCodeHashes publishes code and codehash pairs to the ipld database
func (sdt *StateDiffTransformer) processCodeAndCodeHashes(tx *sqlx.Tx, codeAndCodeHashes []sdtypes.CodeAndCodeHash) error {
	for _, c := range codeAndCodeHashes {
		// codec doesn't matter since db key is multihash-based
		mhKey, err := shared.MultihashKeyFromKeccak256(c.Hash)
		if err != nil {
			return err
		}
		if err := shared.PublishDirect(tx, mhKey, c.Code); err != nil {
			return err
		}
	}
	return nil
}

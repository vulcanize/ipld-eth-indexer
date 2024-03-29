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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"

	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/testhelpers"
	sdtypes "github.com/ethereum/go-ethereum/statediff/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
	"github.com/vulcanize/ipld-eth-indexer/pkg/ipfs/ipld"
)

// Test variables
var (
	// block data
	BlockNumber = big.NewInt(1)
	MockHeader  = types.Header{
		Time:        0,
		Number:      new(big.Int).Set(BlockNumber),
		Root:        common.HexToHash("0x0"),
		TxHash:      common.HexToHash("0x0"),
		ReceiptHash: common.HexToHash("0x0"),
		Difficulty:  big.NewInt(5000000),
		Extra:       []byte{},
	}
	MockTransactions, MockReceipts, SenderAddr = createTransactionsAndReceipts()
	ReceiptsRlp, _                             = rlp.EncodeToBytes(MockReceipts)
	MockBlock                                  = types.NewBlock(&MockHeader, MockTransactions, nil, MockReceipts, new(trie.Trie))
	MockBlockRlp, _                            = rlp.EncodeToBytes(MockBlock)
	MockHeaderRlp, _                           = rlp.EncodeToBytes(MockBlock.Header())
	Address                                    = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476592")
	AnotherAddress                             = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476593")
	ContractAddress                            = crypto.CreateAddress(SenderAddr, MockTransactions[2].Nonce())
	ContractHash                               = crypto.Keccak256Hash(ContractAddress.Bytes()).String()
	MockContractByteCode                       = []byte{0, 1, 2, 3, 4, 5}
	MockCodeHash                               = crypto.Keccak256Hash(MockContractByteCode)
	mockTopic11                                = common.HexToHash("0x04")
	mockTopic12                                = common.HexToHash("0x06")
	mockTopic21                                = common.HexToHash("0x05")
	mockTopic22                                = common.HexToHash("0x07")
	MockLog1                                   = &types.Log{
		Address: Address,
		Topics:  []common.Hash{mockTopic11, mockTopic12},
		Data:    []byte{},
	}
	MockLog2 = &types.Log{
		Address: AnotherAddress,
		Topics:  []common.Hash{mockTopic21, mockTopic22},
		Data:    []byte{},
	}
	Tx1 = GetTxnRlp(0, MockTransactions)
	Tx2 = GetTxnRlp(1, MockTransactions)
	Tx3 = GetTxnRlp(2, MockTransactions)

	Rct1 = GetRctRlp(0, MockReceipts)
	Rct2 = GetRctRlp(1, MockReceipts)
	Rct3 = GetRctRlp(2, MockReceipts)

	HeaderCID, _  = ipld.RawdataToCid(ipld.MEthHeader, MockHeaderRlp, multihash.KECCAK_256)
	HeaderMhKey   = shared.MultihashKeyFromCID(HeaderCID)
	Trx1CID, _    = ipld.RawdataToCid(ipld.MEthTx, Tx1, multihash.KECCAK_256)
	Trx1MhKey     = shared.MultihashKeyFromCID(Trx1CID)
	Trx2CID, _    = ipld.RawdataToCid(ipld.MEthTx, Tx2, multihash.KECCAK_256)
	Trx2MhKey     = shared.MultihashKeyFromCID(Trx2CID)
	Trx3CID, _    = ipld.RawdataToCid(ipld.MEthTx, Tx3, multihash.KECCAK_256)
	Trx3MhKey     = shared.MultihashKeyFromCID(Trx3CID)
	Rct1CID, _    = ipld.RawdataToCid(ipld.MEthTxReceipt, Rct1, multihash.KECCAK_256)
	Rct1MhKey     = shared.MultihashKeyFromCID(Rct1CID)
	Rct2CID, _    = ipld.RawdataToCid(ipld.MEthTxReceipt, Rct2, multihash.KECCAK_256)
	Rct2MhKey     = shared.MultihashKeyFromCID(Rct2CID)
	Rct3CID, _    = ipld.RawdataToCid(ipld.MEthTxReceipt, Rct3, multihash.KECCAK_256)
	Rct3MhKey     = shared.MultihashKeyFromCID(Rct3CID)
	State1CID, _  = ipld.RawdataToCid(ipld.MEthStateTrie, ContractLeafNode, multihash.KECCAK_256)
	State1MhKey   = shared.MultihashKeyFromCID(State1CID)
	State2CID, _  = ipld.RawdataToCid(ipld.MEthStateTrie, AccountLeafNode, multihash.KECCAK_256)
	State2MhKey   = shared.MultihashKeyFromCID(State2CID)
	StorageCID, _ = ipld.RawdataToCid(ipld.MEthStorageTrie, StorageLeafNode, multihash.KECCAK_256)
	StorageMhKey  = shared.MultihashKeyFromCID(StorageCID)
	MockTrxMeta   = []eth.TxModel{
		{
			CID:    "", // This is empty until we go to publish to ipfs
			MhKey:  "",
			Src:    SenderAddr.Hex(),
			Dst:    Address.String(),
			Index:  0,
			TxHash: MockTransactions[0].Hash().String(),
			Data:   []byte{},
		},
		{
			CID:    "",
			MhKey:  "",
			Src:    SenderAddr.Hex(),
			Dst:    AnotherAddress.String(),
			Index:  1,
			TxHash: MockTransactions[1].Hash().String(),
			Data:   []byte{},
		},
		{
			CID:    "",
			MhKey:  "",
			Src:    SenderAddr.Hex(),
			Dst:    "",
			Index:  2,
			TxHash: MockTransactions[2].Hash().String(),
			Data:   MockContractByteCode,
		},
	}
	MockTrxMetaPostPublsh = []eth.TxModel{
		{
			CID:    Trx1CID.String(), // This is empty until we go to publish to ipfs
			MhKey:  Trx1MhKey,
			Src:    SenderAddr.Hex(),
			Dst:    Address.String(),
			Index:  0,
			TxHash: MockTransactions[0].Hash().String(),
			Data:   []byte{},
		},
		{
			CID:    Trx2CID.String(),
			MhKey:  Trx2MhKey,
			Src:    SenderAddr.Hex(),
			Dst:    AnotherAddress.String(),
			Index:  1,
			TxHash: MockTransactions[1].Hash().String(),
			Data:   []byte{},
		},
		{
			CID:    Trx3CID.String(),
			MhKey:  Trx3MhKey,
			Src:    SenderAddr.Hex(),
			Dst:    "",
			Index:  2,
			TxHash: MockTransactions[2].Hash().String(),
			Data:   MockContractByteCode,
		},
	}
	MockRctMeta = []eth.ReceiptModel{
		{
			CID:        "",
			MhKey:      "",
			PostStatus: 1,
			Topic0s: []string{
				mockTopic11.String(),
			},
			Topic1s: []string{
				mockTopic12.String(),
			},
			Contract:     "",
			ContractHash: "",
			LogContracts: []string{
				Address.String(),
			},
		},
		{
			CID:       "",
			MhKey:     "",
			PostState: common.Bytes2Hex(common.HexToHash("0x1").Bytes()),
			Topic0s: []string{
				mockTopic21.String(),
			},
			Topic1s: []string{
				mockTopic22.String(),
			},
			Contract:     "",
			ContractHash: "",
			LogContracts: []string{
				AnotherAddress.String(),
			},
		},
		{
			CID:          "",
			MhKey:        "",
			PostState:    common.Bytes2Hex(common.HexToHash("0x2").Bytes()),
			Contract:     ContractAddress.String(),
			ContractHash: ContractHash,
			LogContracts: []string{},
		},
	}
	MockRctMetaPostPublish = []eth.ReceiptModel{
		{
			CID:        Rct1CID.String(),
			MhKey:      Rct1MhKey,
			PostStatus: 1,
			Topic0s: []string{
				mockTopic11.String(),
			},
			Topic1s: []string{
				mockTopic12.String(),
			},
			Contract:     "",
			ContractHash: "",
			LogContracts: []string{
				Address.String(),
			},
		},
		{
			CID:       Rct2CID.String(),
			MhKey:     Rct2MhKey,
			PostState: common.Bytes2Hex(common.HexToHash("0x1").Bytes()),
			Topic0s: []string{
				mockTopic21.String(),
			},
			Topic1s: []string{
				mockTopic22.String(),
			},
			Contract:     "",
			ContractHash: "",
			LogContracts: []string{
				AnotherAddress.String(),
			},
		},
		{
			CID:          Rct3CID.String(),
			MhKey:        Rct3MhKey,
			PostState:    common.Bytes2Hex(common.HexToHash("0x2").Bytes()),
			Contract:     ContractAddress.String(),
			ContractHash: ContractHash,
			LogContracts: []string{},
		},
	}

	// statediff data
	storageLocation    = common.HexToHash("0")
	StorageLeafKey     = crypto.Keccak256Hash(storageLocation[:]).Bytes()
	StorageValue       = common.Hex2Bytes("01")
	StoragePartialPath = common.Hex2Bytes("20290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	StorageLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		StoragePartialPath,
		StorageValue,
	})

	nonce1             = uint64(1)
	ContractRoot       = "0x821e2556a290c86405f8160a2d662042a431ba456b9db265c79bb837c04be5f0"
	ContractCodeHash   = common.HexToHash("0x753f98a8d4328b15636e46f66f2cb4bc860100aa17967cc145fcd17d1d4710ea")
	contractPath       = common.Bytes2Hex([]byte{'\x06'})
	ContractLeafKey    = testhelpers.AddressToLeafKey(ContractAddress)
	ContractAccount, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    nonce1,
		Balance:  big.NewInt(0),
		CodeHash: ContractCodeHash.Bytes(),
		Root:     common.HexToHash(ContractRoot),
	})
	ContractPartialPath = common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45")
	ContractLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		ContractPartialPath,
		ContractAccount,
	})

	nonce0          = uint64(0)
	AccountRoot     = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
	AccountCodeHash = common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	accountPath     = common.Bytes2Hex([]byte{'\x0c'})
	AccountAddresss = common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e")
	AccountLeafKey  = testhelpers.Account2LeafKey
	Account, _      = rlp.EncodeToBytes(state.Account{
		Nonce:    nonce0,
		Balance:  big.NewInt(1000),
		CodeHash: AccountCodeHash.Bytes(),
		Root:     common.HexToHash(AccountRoot),
	})
	AccountPartialPath = common.Hex2Bytes("3957f3e2f04a0764c3a0491b175f69926da61efbcc8f61fa1455fd2d2b4cdd45")
	AccountLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		AccountPartialPath,
		Account,
	})

	StateDiffs = []sdtypes.StateNode{
		{
			Path:      []byte{'\x06'},
			NodeType:  sdtypes.Leaf,
			LeafKey:   ContractLeafKey,
			NodeValue: ContractLeafNode,
			StorageNodes: []sdtypes.StorageNode{
				{
					Path:      []byte{},
					NodeType:  sdtypes.Leaf,
					LeafKey:   StorageLeafKey,
					NodeValue: StorageLeafNode,
				},
			},
		},
		{
			Path:         []byte{'\x0c'},
			NodeType:     sdtypes.Leaf,
			LeafKey:      AccountLeafKey,
			NodeValue:    AccountLeafNode,
			StorageNodes: []sdtypes.StorageNode{},
		},
	}

	MockStateDiff = statediff.StateObject{
		BlockNumber: new(big.Int).Set(BlockNumber),
		BlockHash:   MockBlock.Hash(),
		Nodes:       StateDiffs,
		CodeAndCodeHashes: []sdtypes.CodeAndCodeHash{
			{
				Code: MockContractByteCode,
				Hash: MockCodeHash,
			},
		},
	}
	MockStateDiffBytes, _ = rlp.EncodeToBytes(MockStateDiff)
	MockStateNodes        = []eth.TrieNode{
		{
			LeafKey: common.BytesToHash(ContractLeafKey),
			Path:    []byte{'\x06'},
			Value:   ContractLeafNode,
			Type:    sdtypes.Leaf,
		},
		{
			LeafKey: common.BytesToHash(AccountLeafKey),
			Path:    []byte{'\x0c'},
			Value:   AccountLeafNode,
			Type:    sdtypes.Leaf,
		},
	}
	MockStateMetaPostPublish = []eth.StateNodeModel{
		{
			CID:      State1CID.String(),
			MhKey:    State1MhKey,
			Path:     []byte{'\x06'},
			NodeType: 2,
			StateKey: common.BytesToHash(ContractLeafKey).Hex(),
		},
		{
			CID:      State2CID.String(),
			MhKey:    State2MhKey,
			Path:     []byte{'\x0c'},
			NodeType: 2,
			StateKey: common.BytesToHash(AccountLeafKey).Hex(),
		},
	}
	MockStorageNodes = map[string][]eth.TrieNode{
		contractPath: {
			{
				LeafKey: common.BytesToHash(StorageLeafKey),
				Value:   StorageLeafNode,
				Type:    sdtypes.Leaf,
				Path:    []byte{},
			},
		},
	}

	// aggregate payloads
	MockStateDiffPayload = statediff.Payload{
		BlockRlp:        MockBlockRlp,
		StateObjectRlp:  MockStateDiffBytes,
		ReceiptsRlp:     ReceiptsRlp,
		TotalDifficulty: MockBlock.Difficulty(),
	}

	MockConvertedPayload = eth.ConvertedPayload{
		TotalDifficulty: MockBlock.Difficulty(),
		Block:           MockBlock,
		Receipts:        MockReceipts,
		TxMetaData:      MockTrxMeta,
		ReceiptMetaData: MockRctMeta,
		StorageNodes:    MockStorageNodes,
		StateNodes:      MockStateNodes,
	}

	MockCIDPayload = eth.CIDPayload{
		HeaderCID: eth.HeaderModel{
			BlockHash:       MockBlock.Hash().String(),
			BlockNumber:     MockBlock.Number().String(),
			CID:             HeaderCID.String(),
			MhKey:           HeaderMhKey,
			ParentHash:      MockBlock.ParentHash().String(),
			TotalDifficulty: MockBlock.Difficulty().String(),
			Reward:          "5000000000000000000",
			StateRoot:       MockBlock.Root().String(),
			RctRoot:         MockBlock.ReceiptHash().String(),
			TxRoot:          MockBlock.TxHash().String(),
			UncleRoot:       MockBlock.UncleHash().String(),
			Bloom:           MockBlock.Bloom().Bytes(),
			Timestamp:       MockBlock.Time(),
		},
		UncleCIDs:       []eth.UncleModel{},
		TransactionCIDs: MockTrxMetaPostPublsh,
		ReceiptCIDs: map[common.Hash]eth.ReceiptModel{
			MockTransactions[0].Hash(): MockRctMetaPostPublish[0],
			MockTransactions[1].Hash(): MockRctMetaPostPublish[1],
			MockTransactions[2].Hash(): MockRctMetaPostPublish[2],
		},
		StateNodeCIDs: MockStateMetaPostPublish,
		StorageNodeCIDs: map[string][]eth.StorageNodeModel{
			contractPath: {
				{
					CID:        StorageCID.String(),
					MhKey:      StorageMhKey,
					Path:       []byte{},
					StorageKey: common.BytesToHash(StorageLeafKey).Hex(),
					NodeType:   2,
				},
			},
		},
		StateAccounts: map[string]eth.StateAccountModel{
			contractPath: {
				Balance:     big.NewInt(0).String(),
				Nonce:       nonce1,
				CodeHash:    ContractCodeHash.Bytes(),
				StorageRoot: common.HexToHash(ContractRoot).String(),
			},
			accountPath: {
				Balance:     big.NewInt(1000).String(),
				Nonce:       nonce0,
				CodeHash:    AccountCodeHash.Bytes(),
				StorageRoot: common.HexToHash(AccountRoot).String(),
			},
		},
	}
)

// createTransactionsAndReceipts is a helper function to generate signed mock transactions and mock receipts with mock logs
func createTransactionsAndReceipts() (types.Transactions, types.Receipts, common.Address) {
	// make transactions
	trx1 := types.NewTransaction(0, Address, big.NewInt(1000), 50, big.NewInt(100), []byte{})
	trx2 := types.NewTransaction(1, AnotherAddress, big.NewInt(2000), 100, big.NewInt(200), []byte{})
	trx3 := types.NewContractCreation(2, big.NewInt(1500), 75, big.NewInt(150), MockContractByteCode)
	transactionSigner := types.MakeSigner(params.MainnetChainConfig, new(big.Int).Set(BlockNumber))
	mockCurve := elliptic.P256()
	mockPrvKey, err := ecdsa.GenerateKey(mockCurve, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	signedTrx1, err := types.SignTx(trx1, transactionSigner, mockPrvKey)
	if err != nil {
		log.Fatal(err)
	}
	signedTrx2, err := types.SignTx(trx2, transactionSigner, mockPrvKey)
	if err != nil {
		log.Fatal(err)
	}
	signedTrx3, err := types.SignTx(trx3, transactionSigner, mockPrvKey)
	if err != nil {
		log.Fatal(err)
	}
	SenderAddr, err := types.Sender(transactionSigner, signedTrx1) // same for both trx
	if err != nil {
		log.Fatal(err)
	}
	// make receipts
	mockReceipt1 := types.NewReceipt(nil, false, 50)
	mockReceipt1.Logs = []*types.Log{MockLog1}
	mockReceipt1.TxHash = signedTrx1.Hash()
	mockReceipt2 := types.NewReceipt(common.HexToHash("0x1").Bytes(), false, 100)
	mockReceipt2.Logs = []*types.Log{MockLog2}
	mockReceipt2.TxHash = signedTrx2.Hash()
	mockReceipt3 := types.NewReceipt(common.HexToHash("0x2").Bytes(), false, 75)
	mockReceipt3.Logs = []*types.Log{}
	mockReceipt3.TxHash = signedTrx3.Hash()
	return types.Transactions{signedTrx1, signedTrx2, signedTrx3}, types.Receipts{mockReceipt1, mockReceipt2, mockReceipt3}, SenderAddr
}

func GetTxnRlp(num int, txs types.Transactions) []byte {
	buf := new(bytes.Buffer)
	txs.EncodeIndex(num, buf)
	tx := make([]byte, buf.Len())
	copy(tx, buf.Bytes())
	buf.Reset()
	return tx
}

func GetRctRlp(num int, rcts types.Receipts) []byte {
	buf := new(bytes.Buffer)
	rcts.EncodeIndex(num, buf)
	rct := make([]byte, buf.Len())
	copy(rct, buf.Bytes())
	buf.Reset()
	return rct
}

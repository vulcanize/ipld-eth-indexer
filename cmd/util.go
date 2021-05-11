// Copyright © 2020 Vulcanize, Inc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
	v "github.com/vulcanize/ipld-eth-indexer/version"
)

// utilCmd represents the backfill command
var utilCmd = &cobra.Command{
	Use:   "util",
	Short: "Util used to write block data to RLP files",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		subCommand = cmd.CalledAs()
		logWithCommand = *log.WithField("SubCommand", subCommand)
		utilCmdCommand()
	},
}

func utilCmdCommand() {
	logWithCommand.Infof("running ipld-eth-indexer version: %s", v.VersionWithMeta)
	start := viper.GetInt64("util.start")
	end := viper.GetInt64("util.end")
	writeDir := viper.GetString("util.dir")
	ethHTTP := viper.GetString("ethereum.httpPath")
	prefix := viper.GetString("util.prefix")
	if prefix != "" {
		if !strings.HasSuffix("prefix", "-") {
			prefix = prefix + "-"
		}
	}
	if _, err := os.Stat(writeDir); os.IsNotExist(err) {
		os.Mkdir(writeDir, 0700)
	}
	_, rpcClient, err := shared.GetEthNodeAndClient(fmt.Sprintf("http://%s", ethHTTP))
	if err != nil {
		logWithCommand.Fatal(err)
	}
	ethClient := ethclient.NewClient(rpcClient)
	for i := start; i <= end; i++ {
		logWithCommand.Infof("writing data for block %d", i)
		block, err := ethClient.BlockByNumber(context.Background(), new(big.Int).SetInt64(i))
		if err != nil {
			logWithCommand.Errorf("error fetching block %d, err: %v", i, err)
			continue
		}
		blockBytes, err := rlp.EncodeToBytes(block)
		if err != nil {
			logWithCommand.Errorf("error RLP encoding block %d, err: %v", i, err)
			continue
		}
		blockFileName := filepath.Join(writeDir, fmt.Sprintf("%seth-block-%d", prefix, i))
		if err := ioutil.WriteFile(blockFileName, blockBytes, 0644); err != nil {
			logWithCommand.Errorf("error writing file for block %d, err %v", i, err)
			continue
		}
		txs := block.Transactions()
		receipts := make(types.Receipts, len(txs))
		for j, tx := range txs {
			fetchedTx, err := ethClient.TransactionInBlock(context.Background(), block.Hash(), uint(j))
			if err != nil {
				logWithCommand.Errorf("error fecthing tx by blockhash and index tx for block %d tx hash %s, err %v", i, tx.Hash().Hex(), err)
			}
			if !bytes.Equal(fetchedTx.Hash().Bytes(), tx.Hash().Bytes()) {
				logWithCommand.Errorf("tx hash %s fetched by block and index does not match known tx hash %s", fetchedTx.Hash().Hex(), tx.Hash().Hex())
			}
			_, _, err = ethClient.TransactionByHash(context.Background(), tx.Hash())
			if err != nil {
				logWithCommand.Errorf("error fetching tx for block %d tx hash %s, err: %v", i, tx.Hash().Hex(), err)
			}
			receipt, err := ethClient.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				logWithCommand.Errorf("error fetching receipt for block %d tx %s, err: %v", i, tx.Hash().Hex(), err)
				continue
			}
			receipts[j] = receipt
		}
		receiptsBytes, err := rlp.EncodeToBytes(receipts)
		if err != nil {
			logWithCommand.Errorf("error RLP encoding receipts for block %d, err %v", i, err)
		}
		receiptsFileName := filepath.Join(writeDir, fmt.Sprintf("%seth-receipts-%d", prefix, i))
		if err := ioutil.WriteFile(receiptsFileName, receiptsBytes, 0644); err != nil {
			logWithCommand.Errorf("error writing file for receipts at block %d, err %v", i, err)
		}
	}
}

func init() {
	rootCmd.AddCommand(utilCmd)

	// flags
	utilCmd.PersistentFlags().String("prefix", "", "prefix for output files, for preventing collisions when comparing results")
	utilCmd.PersistentFlags().String("write-dir", "", "directory to write RLP to")
	utilCmd.PersistentFlags().Int64("block-range-start", 0, "start of the block range")
	utilCmd.PersistentFlags().Int64("block-range-end", -1, "end of the block range")
	utilCmd.PersistentFlags().String("eth-http-path", "", "http url for ethereum node")

	// and their .toml config bindings
	viper.BindPFlag("util.prefix", utilCmd.PersistentFlags().Lookup("prefix"))
	viper.BindPFlag("util.dir", utilCmd.PersistentFlags().Lookup("write-dir"))
	viper.BindPFlag("util.start", utilCmd.PersistentFlags().Lookup("block-range-start"))
	viper.BindPFlag("util.end", utilCmd.PersistentFlags().Lookup("block-range-end"))
	viper.BindPFlag("ethereum.httpPath", utilCmd.PersistentFlags().Lookup("eth-http-path"))
}

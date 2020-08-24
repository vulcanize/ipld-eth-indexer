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
	"fmt"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// backfillCmd represents the backfill command
var backfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("backfill called")
	},
}

func init() {
	rootCmd.AddCommand(backfillCmd)

	// flags for all config variables
	backfillCmd.PersistentFlags().Int("backfill-frequency", 0, "how often (in seconds) the backfill process checks for gaps")
	backfillCmd.PersistentFlags().Int("backfill-batch-size", 0, "data fetching batch size")
	backfillCmd.PersistentFlags().Int("backfill-workers", 0, "how many goroutines to fetch data concurrently")
	backfillCmd.PersistentFlags().Int("backfill-validation-level", 0, "backfill will rebackfill any data below this level")
	backfillCmd.PersistentFlags().Int("backfill-timeout", 0, "timeout used for backfill http requests")

	backfillCmd.PersistentFlags().String("eth-http-path", "", "http url for ethereum node")
	backfillCmd.PersistentFlags().String("eth-node-id", "", "eth node id")
	backfillCmd.PersistentFlags().String("eth-client-name", "", "eth client name")
	backfillCmd.PersistentFlags().String("eth-genesis-block", "", "eth genesis block hash")
	backfillCmd.PersistentFlags().String("eth-network-id", "", "eth network id")
	backfillCmd.PersistentFlags().String("eth-chain-id", "", "eth chain id")

	// and their bindings
	viper.BindPFlag("backfill.frequency", backfillCmd.PersistentFlags().Lookup("backfill-frequency"))
	viper.BindPFlag("backfill.batchSize", backfillCmd.PersistentFlags().Lookup("backfill-batch-size"))
	viper.BindPFlag("backfill.workers", backfillCmd.PersistentFlags().Lookup("backfill-workers"))
	viper.BindPFlag("backfill.validationLevel", backfillCmd.PersistentFlags().Lookup("backfill-validation-level"))
	viper.BindPFlag("backfill.timeout", backfillCmd.PersistentFlags().Lookup("backfill-timeout"))

	viper.BindPFlag("ethereum.httpPath", backfillCmd.PersistentFlags().Lookup("eth-http-path"))
	viper.BindPFlag("ethereum.nodeID", backfillCmd.PersistentFlags().Lookup("eth-node-id"))
	viper.BindPFlag("ethereum.clientName", backfillCmd.PersistentFlags().Lookup("eth-client-name"))
	viper.BindPFlag("ethereum.genesisBlock", backfillCmd.PersistentFlags().Lookup("eth-genesis-block"))
	viper.BindPFlag("ethereum.networkID", backfillCmd.PersistentFlags().Lookup("eth-network-id"))
	viper.BindPFlag("ethereum.chainID", backfillCmd.PersistentFlags().Lookup("eth-chain-id"))
}

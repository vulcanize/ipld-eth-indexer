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
	"os"
	"os/signal"
	s "sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	w "github.com/vulcanize/ipfs-blockchain-watcher/pkg/sync"
	v "github.com/vulcanize/ipfs-blockchain-watcher/version"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync chain data into PG-IPFS",
	Long: `This command configures a VulcanizeDB ipfs-blockchain-watcher.

The Sync process streams all chain data from the appropriate chain, processes this data into IPLD objects
and publishes them to IPFS. It then indexes the CIDs against useful data fields/metadata in Postgres. 

The Serve process creates and exposes a rpc subscription server over ws and ipc. Transformers can subscribe to
these endpoints to stream

The BackFill process spins up a background process which periodically probes the Postgres database to identify
and fill in gaps in the data
`,
	Run: func(cmd *cobra.Command, args []string) {
		subCommand = cmd.CalledAs()
		logWithCommand = *log.WithField("SubCommand", subCommand)
		sync()
	},
}

func sync() {
	logWithCommand.Infof("running ipfs-blockchain-watcher version: %s", v.VersionWithMeta)

	wg := new(s.WaitGroup)
	logWithCommand.Debug("loading configuration variables")
	syncerConfig, err := w.NewConfig()
	if err != nil {
		logWithCommand.Fatal(err)
	}
	logWithCommand.Infof("config: %+v", syncerConfig)
	logWithCommand.Debug("initializing new indexing service")
	syncer, err := w.NewIndexerService(syncerConfig)
	if err != nil {
		logWithCommand.Fatal(err)
	}

	logWithCommand.Info("starting up sync process")
	if err := syncer.Sync(wg); err != nil {
		logWithCommand.Fatal(err)
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt)
	<-shutdown
	syncer.Stop()
	wg.Wait()
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// flags for all config variables
	syncCmd.PersistentFlags().Int("sync-workers", 0, "how many worker goroutines to publish and index data")

	syncCmd.PersistentFlags().String("eth-ws-path", "", "ws url for ethereum node")
	syncCmd.PersistentFlags().String("eth-node-id", "", "eth node id")
	syncCmd.PersistentFlags().String("eth-client-name", "", "eth client name")
	syncCmd.PersistentFlags().String("eth-genesis-block", "", "eth genesis block hash")
	syncCmd.PersistentFlags().String("eth-network-id", "", "eth network id")
	syncCmd.PersistentFlags().String("eth-chain-id", "", "eth chain id")

	// and their bindings
	viper.BindPFlag("sync.workers", syncCmd.PersistentFlags().Lookup("sync-workers"))

	viper.BindPFlag("ethereum.wsPath", syncCmd.PersistentFlags().Lookup("eth-ws-path"))
	viper.BindPFlag("ethereum.nodeID", syncCmd.PersistentFlags().Lookup("eth-node-id"))
	viper.BindPFlag("ethereum.clientName", syncCmd.PersistentFlags().Lookup("eth-client-name"))
	viper.BindPFlag("ethereum.genesisBlock", syncCmd.PersistentFlags().Lookup("eth-genesis-block"))
	viper.BindPFlag("ethereum.networkID", syncCmd.PersistentFlags().Lookup("eth-network-id"))
	viper.BindPFlag("ethereum.chainID", syncCmd.PersistentFlags().Lookup("eth-chain-id"))
}

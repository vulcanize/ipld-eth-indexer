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

	"github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	h "github.com/vulcanize/ipfs-blockchain-watcher/pkg/historical"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/ipfs"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/shared"
	w "github.com/vulcanize/ipfs-blockchain-watcher/pkg/watch"
	v "github.com/vulcanize/ipfs-blockchain-watcher/version"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
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
		watch()
	},
}

func watch() {
	logWithCommand.Infof("running ipfs-blockchain-watcher version: %s", v.VersionWithMeta)

	var forwardPayloadChan chan shared.ConvertedData
	wg := new(s.WaitGroup)
	logWithCommand.Debug("loading watcher configuration variables")
	watcherConfig, err := w.NewConfig()
	if err != nil {
		logWithCommand.Fatal(err)
	}
	logWithCommand.Infof("watcher config: %+v", watcherConfig)
	if watcherConfig.IPFSMode == shared.LocalInterface {
		if err := ipfs.InitIPFSPlugins(); err != nil {
			logWithCommand.Fatal(err)
		}
	}
	logWithCommand.Debug("initializing new watcher service")
	watcher, err := w.NewWatcher(watcherConfig)
	if err != nil {
		logWithCommand.Fatal(err)
	}

	if watcherConfig.Serve {
		logWithCommand.Info("starting up watcher servers")
		forwardPayloadChan = make(chan shared.ConvertedData, w.PayloadChanBufferSize)
		watcher.Serve(wg, forwardPayloadChan)
		if err := startServers(watcher, watcherConfig); err != nil {
			logWithCommand.Fatal(err)
		}
	}

	if watcherConfig.Sync {
		logWithCommand.Info("starting up watcher sync process")
		if err := watcher.Sync(wg, forwardPayloadChan); err != nil {
			logWithCommand.Fatal(err)
		}
	}

	var backFiller h.BackFillInterface
	if watcherConfig.Historical {
		historicalConfig, err := h.NewConfig()
		if err != nil {
			logWithCommand.Fatal(err)
		}
		logWithCommand.Debug("initializing new historical backfill service")
		backFiller, err = h.NewBackFillService(historicalConfig, forwardPayloadChan)
		if err != nil {
			logWithCommand.Fatal(err)
		}
		logWithCommand.Info("starting up watcher backfill process")
		backFiller.BackFill(wg)
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt)
	<-shutdown
	if watcherConfig.Historical {
		backFiller.Stop()
	}
	watcher.Stop()
	wg.Wait()
}

func startServers(watcher w.Watcher, settings *w.Config) error {
	logWithCommand.Debug("starting up IPC server")
	_, _, err := rpc.StartIPCEndpoint(settings.IPCEndpoint, watcher.APIs())
	if err != nil {
		return err
	}
	logWithCommand.Debug("starting up WS server")
	_, _, err = rpc.StartWSEndpoint(settings.WSEndpoint, watcher.APIs(), []string{"vdb"}, nil, true)
	if err != nil {
		return err
	}
	logWithCommand.Debug("starting up HTTP server")
	_, _, err = rpc.StartHTTPEndpoint(settings.HTTPEndpoint, watcher.APIs(), []string{settings.Chain.API()}, nil, nil, rpc.HTTPTimeouts{})
	return err
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// flags for all config variables
	watchCmd.PersistentFlags().String("ipfs-path", "", "ipfs repository path")
	watchCmd.PersistentFlags().String("ipfs-mode", "", "ipfs operation mode")

	watchCmd.PersistentFlags().String("watcher-chain", "", "which chain to support, options are currently Ethereum or Bitcoin.")
	watchCmd.PersistentFlags().Bool("watcher-server", false, "turn vdb server on or off")
	watchCmd.PersistentFlags().String("watcher-ws-path", "", "vdb server ws path")
	watchCmd.PersistentFlags().String("watcher-http-path", "", "vdb server http path")
	watchCmd.PersistentFlags().String("watcher-ipc-path", "", "vdb server ipc path")
	watchCmd.PersistentFlags().Bool("watcher-sync", false, "turn vdb sync on or off")
	watchCmd.PersistentFlags().Int("watcher-workers", 0, "how many worker goroutines to publish and index data")
	watchCmd.PersistentFlags().Bool("watcher-back-fill", false, "turn vdb backfill on or off")
	watchCmd.PersistentFlags().Int("watcher-frequency", 0, "how often (in seconds) the backfill process checks for gaps")
	watchCmd.PersistentFlags().Int("watcher-batch-size", 0, "data fetching batch size")
	watchCmd.PersistentFlags().Int("watcher-batch-number", 0, "how many goroutines to fetch data concurrently")
	watchCmd.PersistentFlags().Int("watcher-validation-level", 0, "backfill will resync any data below this level")
	watchCmd.PersistentFlags().Int("watcher-timeout", 0, "timeout used for backfill http requests")

	watchCmd.PersistentFlags().String("btc-ws-path", "", "ws url for bitcoin node")
	watchCmd.PersistentFlags().String("btc-http-path", "", "http url for bitcoin node")
	watchCmd.PersistentFlags().String("btc-password", "", "password for btc node")
	watchCmd.PersistentFlags().String("btc-username", "", "username for btc node")
	watchCmd.PersistentFlags().String("btc-node-id", "", "btc node id")
	watchCmd.PersistentFlags().String("btc-client-name", "", "btc client name")
	watchCmd.PersistentFlags().String("btc-genesis-block", "", "btc genesis block hash")
	watchCmd.PersistentFlags().String("btc-network-id", "", "btc network id")

	watchCmd.PersistentFlags().String("eth-ws-path", "", "ws url for ethereum node")
	watchCmd.PersistentFlags().String("eth-http-path", "", "http url for ethereum node")
	watchCmd.PersistentFlags().String("eth-node-id", "", "eth node id")
	watchCmd.PersistentFlags().String("eth-client-name", "", "eth client name")
	watchCmd.PersistentFlags().String("eth-genesis-block", "", "eth genesis block hash")
	watchCmd.PersistentFlags().String("eth-network-id", "", "eth network id")

	// and their bindings
	viper.BindPFlag("ipfs.path", watchCmd.PersistentFlags().Lookup("ipfs-path"))
	viper.BindPFlag("ipfs.mode", watchCmd.PersistentFlags().Lookup("ipfs-mode"))

	viper.BindPFlag("watcher.chain", watchCmd.PersistentFlags().Lookup("watcher-chain"))
	viper.BindPFlag("watcher.server", watchCmd.PersistentFlags().Lookup("watcher-server"))
	viper.BindPFlag("watcher.wsPath", watchCmd.PersistentFlags().Lookup("watcher-ws-path"))
	viper.BindPFlag("watcher.httpPath", watchCmd.PersistentFlags().Lookup("watcher-http-path"))
	viper.BindPFlag("watcher.ipcPath", watchCmd.PersistentFlags().Lookup("watcher-ipc-path"))
	viper.BindPFlag("watcher.sync", watchCmd.PersistentFlags().Lookup("watcher-sync"))
	viper.BindPFlag("watcher.workers", watchCmd.PersistentFlags().Lookup("watcher-workers"))
	viper.BindPFlag("watcher.backFill", watchCmd.PersistentFlags().Lookup("watcher-back-fill"))
	viper.BindPFlag("watcher.frequency", watchCmd.PersistentFlags().Lookup("watcher-frequency"))
	viper.BindPFlag("watcher.batchSize", watchCmd.PersistentFlags().Lookup("watcher-batch-size"))
	viper.BindPFlag("watcher.batchNumber", watchCmd.PersistentFlags().Lookup("watcher-batch-number"))
	viper.BindPFlag("watcher.validationLevel", watchCmd.PersistentFlags().Lookup("watcher-validation-level"))
	viper.BindPFlag("watcher.timeout", watchCmd.PersistentFlags().Lookup("watcher-timeout"))

	viper.BindPFlag("bitcoin.wsPath", watchCmd.PersistentFlags().Lookup("btc-ws-path"))
	viper.BindPFlag("bitcoin.httpPath", watchCmd.PersistentFlags().Lookup("btc-http-path"))
	viper.BindPFlag("bitcoin.pass", watchCmd.PersistentFlags().Lookup("btc-password"))
	viper.BindPFlag("bitcoin.user", watchCmd.PersistentFlags().Lookup("btc-username"))
	viper.BindPFlag("bitcoin.nodeID", watchCmd.PersistentFlags().Lookup("btc-node-id"))
	viper.BindPFlag("bitcoin.clientName", watchCmd.PersistentFlags().Lookup("btc-client-name"))
	viper.BindPFlag("bitcoin.genesisBlock", watchCmd.PersistentFlags().Lookup("btc-genesis-block"))
	viper.BindPFlag("bitcoin.networkID", watchCmd.PersistentFlags().Lookup("btc-network-id"))

	viper.BindPFlag("ethereum.wsPath", watchCmd.PersistentFlags().Lookup("eth-ws-path"))
	viper.BindPFlag("ethereum.httpPath", watchCmd.PersistentFlags().Lookup("eth-http-path"))
	viper.BindPFlag("ethereum.nodeID", watchCmd.PersistentFlags().Lookup("eth-node-id"))
	viper.BindPFlag("ethereum.clientName", watchCmd.PersistentFlags().Lookup("eth-client-name"))
	viper.BindPFlag("ethereum.genesisBlock", watchCmd.PersistentFlags().Lookup("eth-genesis-block"))
	viper.BindPFlag("ethereum.networkID", watchCmd.PersistentFlags().Lookup("eth-network-id"))
}

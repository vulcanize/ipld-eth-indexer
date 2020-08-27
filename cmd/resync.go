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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/resync"

	v "github.com/vulcanize/ipfs-blockchain-watcher/version"
)

// resyncCmd represents the resync command
var resyncCmd = &cobra.Command{
	Use:   "resync",
	Short: "Resync historical data",
	Long: `Use this command to define historical block ranges to sync data within
This does not find gaps or under-validated data, it resyncs all the data in the provided range
This can be ran in parallel on non-overlapping regions to scale historical data syncing or
used to force resyncing of data from a new source

NOTE: Requires a syncmode=full gcmode=archive statediffing go-ethereum node`,
	Run: func(cmd *cobra.Command, args []string) {
		subCommand = cmd.CalledAs()
		logWithCommand = *log.WithField("SubCommand", subCommand)
		rsyncCmdCommand()
	},
}

func rsyncCmdCommand() {
	logWithCommand.Infof("running ipld-eth-indexer version: %s", v.VersionWithMeta)
	logWithCommand.Debug("loading resync configuration variables")
	rConfig, err := resync.NewConfig()
	if err != nil {
		logWithCommand.Fatal(err)
	}
	logWithCommand.Infof("resync config: %+v", rConfig)
	logWithCommand.Debug("initializing new resync service")
	rService, err := resync.NewResyncService(rConfig)
	if err != nil {
		logWithCommand.Fatal(err)
	}
	logWithCommand.Info("starting up resync process")
	if err := rService.Sync(); err != nil {
		logWithCommand.Fatal(err)
	}
	logWithCommand.Infof("ethereum %s resync finished", rConfig.ResyncType.String())
}

func init() {
	rootCmd.AddCommand(resyncCmd)

	// flags
	resyncCmd.PersistentFlags().String("resync-type", "", "which type of data to resync")
	resyncCmd.PersistentFlags().Int("resync-start", 0, "block height to start resync")
	resyncCmd.PersistentFlags().Int("resync-stop", 0, "block height to stop resync")
	resyncCmd.PersistentFlags().Int("resync-batch-size", 0, "batch size for http requests")
	resyncCmd.PersistentFlags().Int("resync-workers", 0, "number of worker goroutines to concurrently make and process http requests")
	resyncCmd.PersistentFlags().Bool("resync-clear-old-cache", false, "if true, clear out old data of the provided type within the resync range before resyncing (warning: clearing out data will delete any rows that FK reference it")
	resyncCmd.PersistentFlags().Bool("resync-reset-validation", false, "if true, reset times_validated of headers in this range to 0")
	resyncCmd.PersistentFlags().String("eth-http-path", "", "http url for ethereum node")

	// and their .toml config bindings
	viper.BindPFlag("resync.type", resyncCmd.PersistentFlags().Lookup("resync-type"))
	viper.BindPFlag("resync.start", resyncCmd.PersistentFlags().Lookup("resync-start"))
	viper.BindPFlag("resync.stop", resyncCmd.PersistentFlags().Lookup("resync-stop"))
	viper.BindPFlag("resync.batchSize", resyncCmd.PersistentFlags().Lookup("resync-batch-size"))
	viper.BindPFlag("resync.workers", resyncCmd.PersistentFlags().Lookup("resync-workers"))
	viper.BindPFlag("resync.clearOldCache", resyncCmd.PersistentFlags().Lookup("resync-clear-old-cache"))
	viper.BindPFlag("resync.resetValidation", resyncCmd.PersistentFlags().Lookup("resync-reset-validation"))
	viper.BindPFlag("resync.timeout", resyncCmd.PersistentFlags().Lookup("resync-timeout"))
	viper.BindPFlag("ethereum.httpPath", resyncCmd.PersistentFlags().Lookup("eth-http-path"))
}

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
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/historical"

	v "github.com/vulcanize/ipfs-blockchain-watcher/version"
)

// backfillCmd represents the backfill command
var backfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "Find and fill gaps in database",
	Long: `This command looks for gaps in the vdb postgres database, filling in the missing data
It searches for heights where no header exists and heights where the header has been validated fewer times
than the specified limit

NOTE: Requires a syncmode=full gcmode=archive statediffing go-ethereum node`,
	Run: func(cmd *cobra.Command, args []string) {
		subCommand = cmd.CalledAs()
		logWithCommand = *log.WithField("SubCommand", subCommand)
		backfillCmdCommand()
	},
}

func backfillCmdCommand() {
	logWithCommand.Infof("running ipld-eth-indexer version: %s", v.VersionWithMeta)

	wg := new(s.WaitGroup)
	logWithCommand.Debug("loading backfill configuration variables")
	bConfig, err := historical.NewConfig()
	if err != nil {
		logWithCommand.Fatal(err)
	}
	logWithCommand.Infof("backfill config: %+v", bConfig)
	logWithCommand.Debug("initializing new backfill service")
	bService, err := historical.NewBackfillService(bConfig)
	if err != nil {
		logWithCommand.Fatal(err)
	}
	logWithCommand.Info("starting up backfill process")
	bService.Sync(wg)

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt)
	<-shutdown
	bService.Stop()
	wg.Wait()
}

func init() {
	rootCmd.AddCommand(backfillCmd)

	// flags
	backfillCmd.PersistentFlags().Int("backfill-frequency", 15, "how often to search for new gaps (in seconds; default 15)")
	backfillCmd.PersistentFlags().Int("backfill-batch-size", 2, "batch size for http requests")
	backfillCmd.PersistentFlags().Int("backfill-workers", 4, "number of worker goroutines to concurrently make and process http requests")
	backfillCmd.PersistentFlags().Int("backfill-timeout", 15, "timeout used for backfill http requests (in seconds)")
	backfillCmd.PersistentFlags().Int("backfill-validation-level", 1, "data validated less than this amount will be backfilled")
	backfillCmd.PersistentFlags().String("eth-http-path", "", "http url for ethereum node")

	// and their .toml config bindings
	viper.BindPFlag("backfill.frequency", backfillCmd.PersistentFlags().Lookup("backfill-frequency"))
	viper.BindPFlag("backfill.batchSize", backfillCmd.PersistentFlags().Lookup("backfill-batch-size"))
	viper.BindPFlag("backfill.workers", backfillCmd.PersistentFlags().Lookup("backfill-workers"))
	viper.BindPFlag("backfill.timeout", backfillCmd.PersistentFlags().Lookup("backfill-timeout"))
	viper.BindPFlag("backfill.validationLevel", backfillCmd.PersistentFlags().Lookup("backfill-validation-level"))
	viper.BindPFlag("ethereum.httpPath", backfillCmd.PersistentFlags().Lookup("eth-http-path"))
}

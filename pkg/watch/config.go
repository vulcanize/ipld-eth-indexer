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

package watch

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/viper"

	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/node"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/postgres"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/shared"
	"github.com/vulcanize/ipfs-blockchain-watcher/utils"
)

// Env variables
const (
	WORKERS = "SYNC_WORKERS"

	SYNC_MAX_IDLE_CONNECTIONS = "SYNC_MAX_IDLE_CONNECTIONS"
	SYNC_MAX_OPEN_CONNECTIONS = "SYNC_MAX_OPEN_CONNECTIONS"
	SYNC_MAX_CONN_LIFETIME    = "SYNC_MAX_CONN_LIFETIME"
)

// Config struct
type Config struct {
	DB       *postgres.DB
	DBConfig postgres.Config
	Workers  int
	WSClient *rpc.Client
	NodeInfo node.Info
}

// NewConfig is used to initialize a watcher config from a .toml file
// Separate chain watcher instances need to be ran with separate ipfs path in order to avoid lock contention on the ipfs repository lockfile
func NewConfig() (*Config, error) {
	c := new(Config)
	var err error
	viper.BindEnv("sync.workers", WORKERS)
	viper.BindEnv("ethereum.wsPath", shared.ETH_WS_PATH)
	c.DBConfig.Init()

	workers := viper.GetInt("sync.workers")
	if workers < 1 {
		workers = 1
	}
	c.Workers = workers
	ethWS := viper.GetString("ethereum.wsPath")
	c.NodeInfo, c.WSClient, err = shared.GetEthNodeAndClient(fmt.Sprintf("ws://%s", ethWS))
	if err != nil {
		return nil, err
	}
	syncDBConn := overrideDBConnConfig(c.DBConfig)
	syncDB := utils.LoadPostgres(syncDBConn, c.NodeInfo)
	c.DB = &syncDB

	return c, nil
}

func overrideDBConnConfig(con postgres.Config) postgres.Config {
	viper.BindEnv("database.sync.maxIdle", SYNC_MAX_IDLE_CONNECTIONS)
	viper.BindEnv("database.sync.maxOpen", SYNC_MAX_OPEN_CONNECTIONS)
	viper.BindEnv("database.sync.maxLifetime", SYNC_MAX_CONN_LIFETIME)
	con.MaxIdle = viper.GetInt("database.sync.maxIdle")
	con.MaxOpen = viper.GetInt("database.sync.maxOpen")
	con.MaxLifetime = viper.GetInt("database.sync.maxLifetime")
	return con
}

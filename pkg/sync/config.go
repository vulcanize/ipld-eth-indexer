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

package sync

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/viper"

	"github.com/vulcanize/ipld-eth-indexer/pkg/node"
	"github.com/vulcanize/ipld-eth-indexer/pkg/postgres"
	"github.com/vulcanize/ipld-eth-indexer/pkg/shared"
	"github.com/vulcanize/ipld-eth-indexer/utils"
)

// Env variables
const (
	SYNC_WORKERS = "SYNC_WORKERS"

	SYNC_MAX_IDLE_CONNECTIONS = "SYNC_MAX_IDLE_CONNECTIONS"
	SYNC_MAX_OPEN_CONNECTIONS = "SYNC_MAX_OPEN_CONNECTIONS"
	SYNC_MAX_CONN_LIFETIME    = "SYNC_MAX_CONN_LIFETIME"
)

// Config struct
type Config struct {
	DB       *postgres.DB
	Workers  int64
	WSClient *rpc.Client
	NodeInfo node.Info
}

// NewConfig is used to initialize a sync config from a .toml file
func NewConfig() (*Config, error) {
	c := new(Config)
	var err error
	viper.BindEnv("sync.workers", SYNC_WORKERS)
	viper.BindEnv("ethereum.wsPath", shared.ETH_WS_PATH)

	workers := viper.GetInt64("sync.workers")
	if workers < 1 {
		workers = 1
	}
	c.Workers = workers

	ethWS := viper.GetString("ethereum.wsPath")
	c.NodeInfo, c.WSClient, err = shared.GetEthNodeAndClient(fmt.Sprintf("ws://%s", ethWS))
	if err != nil {
		return nil, err
	}

	dbConfig := postgres.NewConfig()
	overrideDBConnConfig(dbConfig)
	syncDB := utils.LoadPostgres(dbConfig, c.NodeInfo, true)
	c.DB = &syncDB
	return c, nil
}

func overrideDBConnConfig(con *postgres.Config) {
	viper.BindEnv("database.sync.maxIdle", SYNC_MAX_IDLE_CONNECTIONS)
	viper.BindEnv("database.sync.maxOpen", SYNC_MAX_OPEN_CONNECTIONS)
	viper.BindEnv("database.sync.maxLifetime", SYNC_MAX_CONN_LIFETIME)
	con.MaxIdle = viper.GetInt("database.sync.maxIdle")
	con.MaxOpen = viper.GetInt("database.sync.maxOpen")
	con.MaxLifetime = viper.GetInt("database.sync.maxLifetime")
}

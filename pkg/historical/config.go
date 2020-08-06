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

package historical

import (
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/config"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/node"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/postgres"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/shared"
	"github.com/vulcanize/ipfs-blockchain-watcher/utils"
)

// Env variables
const (
	SUPERNODE_CHAIN            = "SUPERNODE_CHAIN"
	SUPERNODE_FREQUENCY        = "SUPERNODE_FREQUENCY"
	SUPERNODE_BATCH_SIZE       = "SUPERNODE_BATCH_SIZE"
	SUPERNODE_BATCH_NUMBER     = "SUPERNODE_BATCH_NUMBER"
	SUPERNODE_VALIDATION_LEVEL = "SUPERNODE_VALIDATION_LEVEL"

	BACKFILL_MAX_IDLE_CONNECTIONS = "BACKFILL_MAX_IDLE_CONNECTIONS"
	BACKFILL_MAX_OPEN_CONNECTIONS = "BACKFILL_MAX_OPEN_CONNECTIONS"
	BACKFILL_MAX_CONN_LIFETIME    = "BACKFILL_MAX_CONN_LIFETIME"
)

// Config struct
type Config struct {
	Chain    shared.ChainType
	IPFSPath string
	IPFSMode shared.IPFSMode
	DBConfig config.Database

	DB              *postgres.DB
	HTTPClient      interface{}
	Frequency       time.Duration
	BatchSize       uint64
	BatchNumber     uint64
	ValidationLevel int
	Timeout         time.Duration // HTTP connection timeout in seconds
	NodeInfo        node.Node
}

// NewConfig is used to initialize a historical config from a .toml file
func NewConfig() (*Config, error) {
	c := new(Config)
	var err error

	viper.BindEnv("watcher.chain", SUPERNODE_CHAIN)
	chain := viper.GetString("watcher.chain")
	c.Chain, err = shared.NewChainType(chain)
	if err != nil {
		return nil, err
	}

	c.IPFSMode, err = shared.GetIPFSMode()
	if err != nil {
		return nil, err
	}
	if c.IPFSMode == shared.LocalInterface || c.IPFSMode == shared.RemoteClient {
		c.IPFSPath, err = shared.GetIPFSPath()
		if err != nil {
			return nil, err
		}
	}

	c.DBConfig.Init()

	if err := c.init(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) init() error {
	var err error

	viper.BindEnv("ethereum.httpPath", shared.ETH_HTTP_PATH)
	viper.BindEnv("bitcoin.httpPath", shared.BTC_HTTP_PATH)
	viper.BindEnv("watcher.frequency", SUPERNODE_FREQUENCY)
	viper.BindEnv("watcher.batchSize", SUPERNODE_BATCH_SIZE)
	viper.BindEnv("watcher.batchNumber", SUPERNODE_BATCH_NUMBER)
	viper.BindEnv("watcher.validationLevel", SUPERNODE_VALIDATION_LEVEL)
	viper.BindEnv("watcher.timeout", shared.HTTP_TIMEOUT)

	timeout := viper.GetInt("watcher.timeout")
	if timeout < 15 {
		timeout = 15
	}
	c.Timeout = time.Second * time.Duration(timeout)

	switch c.Chain {
	case shared.Ethereum:
		ethHTTP := viper.GetString("ethereum.httpPath")
		c.NodeInfo, c.HTTPClient, err = shared.GetEthNodeAndClient(fmt.Sprintf("http://%s", ethHTTP))
		if err != nil {
			return err
		}
	case shared.Bitcoin:
		btcHTTP := viper.GetString("bitcoin.httpPath")
		c.NodeInfo, c.HTTPClient = shared.GetBtcNodeAndClient(btcHTTP)
	}

	freq := viper.GetInt("watcher.frequency")
	var frequency time.Duration
	if freq <= 0 {
		frequency = time.Second * 30
	} else {
		frequency = time.Second * time.Duration(freq)
	}
	c.Frequency = frequency
	c.BatchSize = uint64(viper.GetInt64("watcher.batchSize"))
	c.BatchNumber = uint64(viper.GetInt64("watcher.batchNumber"))
	c.ValidationLevel = viper.GetInt("watcher.validationLevel")

	dbConn := overrideDBConnConfig(c.DBConfig)
	db := utils.LoadPostgres(dbConn, c.NodeInfo)
	c.DB = &db
	return nil
}

func overrideDBConnConfig(con config.Database) config.Database {
	viper.BindEnv("database.backFill.maxIdle", BACKFILL_MAX_IDLE_CONNECTIONS)
	viper.BindEnv("database.backFill.maxOpen", BACKFILL_MAX_OPEN_CONNECTIONS)
	viper.BindEnv("database.backFill.maxLifetime", BACKFILL_MAX_CONN_LIFETIME)
	con.MaxIdle = viper.GetInt("database.backFill.maxIdle")
	con.MaxOpen = viper.GetInt("database.backFill.maxOpen")
	con.MaxLifetime = viper.GetInt("database.backFill.maxLifetime")
	return con
}

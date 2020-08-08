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
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/config"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/node"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/postgres"
	"github.com/vulcanize/ipfs-blockchain-watcher/pkg/shared"
	"github.com/vulcanize/ipfs-blockchain-watcher/utils"
)

// Env variables
const (
	SUPERNODE_CHAIN     = "SUPERNODE_CHAIN"
	SUPERNODE_SYNC      = "SUPERNODE_SYNC"
	SUPERNODE_WORKERS   = "SUPERNODE_WORKERS"
	SUPERNODE_SERVER    = "SUPERNODE_SERVER"
	SUPERNODE_WS_PATH   = "SUPERNODE_WS_PATH"
	SUPERNODE_IPC_PATH  = "SUPERNODE_IPC_PATH"
	SUPERNODE_HTTP_PATH = "SUPERNODE_HTTP_PATH"
	SUPERNODE_BACKFILL  = "SUPERNODE_BACKFILL"

	SYNC_MAX_IDLE_CONNECTIONS = "SYNC_MAX_IDLE_CONNECTIONS"
	SYNC_MAX_OPEN_CONNECTIONS = "SYNC_MAX_OPEN_CONNECTIONS"
	SYNC_MAX_CONN_LIFETIME    = "SYNC_MAX_CONN_LIFETIME"

	SERVER_MAX_IDLE_CONNECTIONS = "SERVER_MAX_IDLE_CONNECTIONS"
	SERVER_MAX_OPEN_CONNECTIONS = "SERVER_MAX_OPEN_CONNECTIONS"
	SERVER_MAX_CONN_LIFETIME    = "SERVER_MAX_CONN_LIFETIME"
)

// Config struct
type Config struct {
	Chain    shared.ChainType
	IPFSPath string
	IPFSMode shared.IPFSMode
	DBConfig config.Database
	// Server fields
	Serve        bool
	ServeDBConn  *postgres.DB
	WSEndpoint   string
	HTTPEndpoint string
	IPCEndpoint  string
	// Sync params
	Sync       bool
	SyncDBConn *postgres.DB
	Workers    int
	WSClient   interface{}
	NodeInfo   node.Node
	// Historical switch
	Historical bool
}

// NewConfig is used to initialize a watcher config from a .toml file
// Separate chain watcher instances need to be ran with separate ipfs path in order to avoid lock contention on the ipfs repository lockfile
func NewConfig() (*Config, error) {
	c := new(Config)
	var err error

	viper.BindEnv("watcher.chain", SUPERNODE_CHAIN)
	viper.BindEnv("watcher.sync", SUPERNODE_SYNC)
	viper.BindEnv("watcher.workers", SUPERNODE_WORKERS)
	viper.BindEnv("ethereum.wsPath", shared.ETH_WS_PATH)
	viper.BindEnv("bitcoin.wsPath", shared.BTC_WS_PATH)
	viper.BindEnv("watcher.server", SUPERNODE_SERVER)
	viper.BindEnv("watcher.wsPath", SUPERNODE_WS_PATH)
	viper.BindEnv("watcher.ipcPath", SUPERNODE_IPC_PATH)
	viper.BindEnv("watcher.httpPath", SUPERNODE_HTTP_PATH)
	viper.BindEnv("watcher.backFill", SUPERNODE_BACKFILL)

	c.Historical = viper.GetBool("watcher.backFill")
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

	c.Sync = viper.GetBool("watcher.sync")
	if c.Sync {
		workers := viper.GetInt("watcher.workers")
		if workers < 1 {
			workers = 1
		}
		c.Workers = workers
		switch c.Chain {
		case shared.Ethereum:
			ethWS := viper.GetString("ethereum.wsPath")
			c.NodeInfo, c.WSClient, err = shared.GetEthNodeAndClient(fmt.Sprintf("ws://%s", ethWS))
			if err != nil {
				return nil, err
			}
		case shared.Bitcoin:
			btcWS := viper.GetString("bitcoin.wsPath")
			c.NodeInfo, c.WSClient = shared.GetBtcNodeAndClient(btcWS)
		}
		syncDBConn := overrideDBConnConfig(c.DBConfig, Sync)
		syncDB := utils.LoadPostgres(syncDBConn, c.NodeInfo)
		c.SyncDBConn = &syncDB
	}

	c.Serve = viper.GetBool("watcher.server")
	if c.Serve {
		wsPath := viper.GetString("watcher.wsPath")
		if wsPath == "" {
			wsPath = "127.0.0.1:8080"
		}
		c.WSEndpoint = wsPath
		ipcPath := viper.GetString("watcher.ipcPath")
		if ipcPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			ipcPath = filepath.Join(home, ".vulcanize/vulcanize.ipc")
		}
		c.IPCEndpoint = ipcPath
		httpPath := viper.GetString("watcher.httpPath")
		if httpPath == "" {
			httpPath = "127.0.0.1:8081"
		}
		c.HTTPEndpoint = httpPath
		serveDBConn := overrideDBConnConfig(c.DBConfig, Serve)
		serveDB := utils.LoadPostgres(serveDBConn, c.NodeInfo)
		c.ServeDBConn = &serveDB
	}

	return c, nil
}

type mode string

var (
	Sync  mode = "sync"
	Serve mode = "serve"
)

func overrideDBConnConfig(con config.Database, m mode) config.Database {
	switch m {
	case Sync:
		viper.BindEnv("database.sync.maxIdle", SYNC_MAX_IDLE_CONNECTIONS)
		viper.BindEnv("database.sync.maxOpen", SYNC_MAX_OPEN_CONNECTIONS)
		viper.BindEnv("database.sync.maxLifetime", SYNC_MAX_CONN_LIFETIME)
		con.MaxIdle = viper.GetInt("database.sync.maxIdle")
		con.MaxOpen = viper.GetInt("database.sync.maxOpen")
		con.MaxLifetime = viper.GetInt("database.sync.maxLifetime")
	case Serve:
		viper.BindEnv("database.server.maxIdle", SERVER_MAX_IDLE_CONNECTIONS)
		viper.BindEnv("database.server.maxOpen", SERVER_MAX_OPEN_CONNECTIONS)
		viper.BindEnv("database.server.maxLifetime", SERVER_MAX_CONN_LIFETIME)
		con.MaxIdle = viper.GetInt("database.server.maxIdle")
		con.MaxOpen = viper.GetInt("database.server.maxOpen")
		con.MaxLifetime = viper.GetInt("database.server.maxLifetime")
	default:
	}
	return con
}

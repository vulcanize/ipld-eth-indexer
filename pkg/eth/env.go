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

package eth

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/viper"

	"github.com/vulcanize/ipld-eth-indexer/pkg/node"
)

// Env variables
const (
	HTTP_TIMEOUT = "HTTP_TIMEOUT"

	ETH_WS_PATH       = "ETH_WS_PATH"
	ETH_HTTP_PATH     = "ETH_HTTP_PATH"
	ETH_NODE_ID       = "ETH_NODE_ID"
	ETH_CLIENT_NAME   = "ETH_CLIENT_NAME"
	ETH_GENESIS_BLOCK = "ETH_GENESIS_BLOCK"
	ETH_NETWORK_ID    = "ETH_NETWORK_ID"
	ETH_CHAIN_ID      = "ETH_CHAIN_ID"
)

// GetNodeAndClient returns eth node info and client from path url
func GetNodeAndClient(path string) (node.Info, *Client, error) {
	viper.BindEnv("ethereum.nodeID", ETH_NODE_ID)
	viper.BindEnv("ethereum.clientName", ETH_CLIENT_NAME)
	viper.BindEnv("ethereum.genesisBlock", ETH_GENESIS_BLOCK)
	viper.BindEnv("ethereum.networkID", ETH_NETWORK_ID)
	viper.BindEnv("ethereum.chainID", ETH_CHAIN_ID)

	rpcClient, err := rpc.Dial(path)
	if err != nil {
		return node.Info{}, nil, err
	}
	return node.Info{
		ID:           viper.GetString("ethereum.nodeID"),
		ClientName:   viper.GetString("ethereum.clientName"),
		GenesisBlock: viper.GetString("ethereum.genesisBlock"),
		NetworkID:    viper.GetString("ethereum.networkID"),
		ChainID:      viper.GetUint64("ethereum.chainID"),
	}, NewClient(rpcClient), nil
}

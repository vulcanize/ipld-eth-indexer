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
	"context"

	"github.com/ethereum/go-ethereum/rpc"
)

// Client is a loose wrapper around the go-ethereum rpc.Client
// It overrides the Subscribe method with a more generic interface for testing purposes
type Client struct {
	*rpc.Client
}

// NewClient returns a new Client
func NewClient(cli *rpc.Client) *Client {
	return &Client{
		Client: cli,
	}
}

// Subscribe method which returns ClientSubscription interface instead of go-ethereum ClientSubscription struct
func (c *Client) Subscribe(ctx context.Context, namespace string, payloadChan interface{}, args ...interface{}) (ClientSubscription, error) {
	return c.Subscribe(ctx, namespace, payloadChan, args...)
}

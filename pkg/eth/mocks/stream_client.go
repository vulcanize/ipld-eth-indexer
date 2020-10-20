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

package mocks

import (
	"context"
	"reflect"

	"github.com/vulcanize/ipld-eth-indexer/pkg/eth"
)

// StreamClient mock eth rpc client
type StreamClient struct {
	StreamPayloads      []interface{}
	PassedContext       context.Context
	PassedResult        interface{}
	PassedNamespace     string
	PassedPayloadChan   interface{}
	PassedSubscribeArgs []interface{}
}

// Subscribe mock method
func (client *StreamClient) Subscribe(ctx context.Context, namespace string, payloadChan interface{}, args ...interface{}) (eth.ClientSubscription, error) {
	client.PassedNamespace = namespace
	client.PassedPayloadChan = payloadChan
	client.PassedContext = ctx

	chanVal := reflect.ValueOf(payloadChan)

	for _, arg := range args {
		client.PassedSubscribeArgs = append(client.PassedSubscribeArgs, arg)
	}
	errChan := make(chan error)
	quitChan := make(chan bool)
	clientSub := &ClientSubscription{
		errChan: errChan,
	}
	go func() {
		for _, payload := range client.StreamPayloads {
			select {
			case <-quitChan:
				return
			default:
			}
			chanVal.Send(reflect.ValueOf(payload))
		}
		close(errChan)
	}()
	return clientSub, nil
}

// ClientSubscription type for testing
type ClientSubscription struct {
	errChan  chan error
	quitChan chan bool
}

// Err satisfies the eth.ClientSubscription interface
func (cs *ClientSubscription) Err() <-chan error {
	return cs.errChan
}

// Unsubscribe satisfies the eth.ClientSubscription interface
func (cs *ClientSubscription) Unsubscribe() {
	close(cs.quitChan)
}

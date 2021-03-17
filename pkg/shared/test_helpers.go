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

package shared

import (
	"os"
	"strconv"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/vulcanize/ipld-eth-indexer/pkg/node"
	"github.com/vulcanize/ipld-eth-indexer/pkg/postgres"
)

// SetupDB is use to setup a db for watcher tests
func SetupDB() (*postgres.DB, error) {
	return postgres.NewDB(getTestConfig(), node.Info{}, true)
}

func SetupDBWithNode(node node.Info) (*postgres.DB, error) {
	return postgres.NewDB(getTestConfig(), node, true)
}

func getTestConfig() *postgres.Config {
	// get connection to test database from environment variables
	hostname := os.Getenv(postgres.DATABASE_HOSTNAME)
	if hostname == "" {
		hostname = "127.0.0.1"
	}
	name := os.Getenv(postgres.DATABASE_NAME)
	if name == "" {
		name = "vulcanize_testing"
	}

	portStr := os.Getenv(postgres.DATABASE_PORT)
	if portStr == "" {
		portStr = "5432"
	}

	port, _ := strconv.Atoi(portStr)

	user := os.Getenv(postgres.DATABASE_USER)
	password := os.Getenv(postgres.DATABASE_PASSWORD)

	return &postgres.Config{
		Hostname: hostname,
		Name:     name,
		Port:     port,
		User:     user,
		Password: password,
	}
}

// ListContainsString used to check if a list of strings contains a particular string
func ListContainsString(sss []string, s string) bool {
	for _, str := range sss {
		if s == str {
			return true
		}
	}
	return false
}

// TestCID creates a basic CID for testing purposes
func TestCID(b []byte) cid.Cid {
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.KECCAK_256,
		MhLength: -1,
	}
	c, _ := pref.Sum(b)
	return c
}

// PublishMockIPLD writes a mhkey-data pair to the public.blocks table so that test data can FK reference the mhkey
func PublishMockIPLD(db *postgres.DB, mhKey string, mockData []byte) error {
	_, err := db.Exec(`INSERT INTO public.blocks (key, data) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`, mhKey, mockData)
	return err
}

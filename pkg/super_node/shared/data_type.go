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
	"fmt"
	"strings"
)

// DataType is an enum to loosely represent type of chain data
type DataType int

const (
	UnknownDataType DataType = iota - 1
	Full
	Headers
	Transactions
	Receipts
	State
	Storage
)

// String() method to resolve ReSyncType enum
func (r DataType) String() string {
	switch r {
	case Full:
		return "full"
	case Headers:
		return "headers"
	case Transactions:
		return "transactions"
	case Receipts:
		return "receipts"
	case State:
		return "state"
	case Storage:
		return "storage"
	default:
		return "unknown"
	}
}

// GenerateResyncTypeFromString
func GenerateResyncTypeFromString(str string) (DataType, error) {
	switch strings.ToLower(str) {
	case "full", "f":
		return Full, nil
	case "headers", "header", "h":
		return Headers, nil
	case "transactions", "transaction", "trxs", "txs", "trx", "tx", "t":
		return Transactions, nil
	case "receipts", "receipt", "rcts", "rct", "r":
		return Receipts, nil
	case "state":
		return State, nil
	case "storage":
		return Storage, nil
	default:
		return UnknownDataType, fmt.Errorf("unrecognized resync type: %s", str)
	}
}

func SupportedResyncType(d DataType) bool {
	switch d {
	case Full:
		return true
	case Headers:
		return false
	case Transactions:
		return false
	case Receipts:
		return false
	case State:
		return false
	case Storage:
		return false
	default:
		return false
	}
}

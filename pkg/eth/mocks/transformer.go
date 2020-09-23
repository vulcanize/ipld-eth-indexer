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
	"github.com/ethereum/go-ethereum/statediff"
)

// Transformer for testing
type Transformer struct {
	PassedWorkerID  int
	PassedStateDiff statediff.Payload
	ReturnHeight    int64
	ReturnErr       error
}

// Transform mock method
func (t *Transformer) Transform(workerID int, payload statediff.Payload) (int64, error) {
	t.PassedWorkerID = workerID
	t.PassedStateDiff = payload
	return t.ReturnHeight, t.ReturnErr
}

// IterativeTransformer for testing
type IterativeTransformer struct {
	PassedWorkerIDs  []int
	PassedStateDiffs []statediff.Payload
	ReturnHeights    []int64
	ReturnErr        error
	iteration        int
}

// Transform mock method
func (t *IterativeTransformer) Transform(workerID int, payload statediff.Payload) (int64, error) {
	t.PassedWorkerIDs = append(t.PassedWorkerIDs, workerID)
	t.PassedStateDiffs = append(t.PassedStateDiffs, payload)
	height := t.ReturnHeights[t.iteration]
	t.iteration++
	return height, t.ReturnErr
}

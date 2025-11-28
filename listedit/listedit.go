//
// Copyright (c) 2025 Canonical Ltd
//
// Original implementation: Gustavo Niemeyer <niemeyer@canonical.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package listedit

import (
	"strconv"
)

type CostInt int64

func (cv CostInt) String() string {
	if cv == Inhibit {
		return "-"
	}
	return strconv.FormatInt(int64(cv), 10)
}

const Inhibit = 1<<63 - 1

type Cost struct {
	SwapAB  CostInt
	DeleteA CostInt
	InsertB CostInt
}

type CostFunc func(ar, br any) Cost

func StandardCost(ar, br any) Cost {
	return Cost{SwapAB: 1, DeleteA: 1, InsertB: 1}
}

type EqualFunc func(ar, br any) bool

func DefaultEqual(ar, br any) bool {
	return ar == br
}

type Op struct {
	Kind   string // The kind of operation, one of "match", "replace", "delete", "insert".
	IdxA   int    // The index of array a where the operation should be applied as the array is being transformed.
	ValueA any    // The value of array a at IdxA as the array is being transformed, relevant for "match", "replace" and "delete" operations.
	IdxB   int    // The index of array b, relevant for "match", "replace" and "insert" operations.
	ValueB any    // The value of array b at IdxB, relevant for "match", "replace" and "insert" operations.
}

func Edit(a, b []any, f CostFunc, eq EqualFunc) (int64, []Op) {
	al, bl := len(a), len(b)

	// Construct the full matrix to extract edit operations.
	matrix := make([][]CostInt, al+1)
	for i := range matrix {
		matrix[i] = make([]CostInt, bl+1)
	}

	// Initialize the first row (insertions).
	matrix[0][0] = 0
	for j := 1; j < bl+1; j++ {
		bj := j - 1
		cost := f(nil, b[bj])
		if cost.InsertB == Inhibit || matrix[0][j-1] == Inhibit {
			matrix[0][j] = Inhibit
		} else {
			matrix[0][j] = matrix[0][j-1] + cost.InsertB
		}
	}

	// Initialize the first column (deletions).
	for i := 1; i < al+1; i++ {
		ai := i - 1
		cost := f(a[ai], nil)
		if cost.DeleteA == Inhibit || matrix[i-1][0] == Inhibit {
			matrix[i][0] = Inhibit
		} else {
			matrix[i][0] = matrix[i-1][0] + cost.DeleteA
		}
	}

	// Fill in the rest of the matrix.
	for i := 1; i < al+1; i++ {
		for j := 1; j < bl+1; j++ {
			ai, bj := i-1, j-1

			cost := f(a[ai], b[bj])
			min := CostInt(Inhibit)

			if eq(a[ai], b[bj]) {
				// Match.
				min = matrix[i-1][j-1]
			} else if cost.SwapAB != Inhibit && matrix[i-1][j-1] != Inhibit {
				// Replace.
				min = matrix[i-1][j-1] + cost.SwapAB
			}

			if cost.DeleteA != Inhibit && matrix[i-1][j] != Inhibit {
				// Delete.
				if n := matrix[i-1][j] + cost.DeleteA; n < min {
					min = n
				}
			}

			if cost.InsertB != Inhibit && matrix[i][j-1] != Inhibit {
				// Insert.
				if n := matrix[i][j-1] + cost.InsertB; n < min {
					min = n
				}
			}

			matrix[i][j] = min
		}
	}

	// Backtrack to extract edit operations.
	ops := []Op{}

	i, j := al, bl
	for i > 0 || j > 0 {
		var cost Cost

		if i > 0 && j > 0 {
			ai, bi := i-1, j-1
			cost = f(a[ai], b[bi])
			if eq(a[ai], b[bi]) && matrix[i][j] == matrix[i-1][j-1] {
				// A match is always the preferred.
				ops = append(ops, Op{Kind: "match", ValueA: a[ai], IdxB: bi, ValueB: b[bi]})
				i--
				j--
				continue
			} else if cost.SwapAB != Inhibit && matrix[i][j] == matrix[i-1][j-1]+cost.SwapAB {
				// A replace is the first preference after match, because it helps aligning the sequences.
				ops = append(ops, Op{Kind: "replace", ValueA: a[ai], IdxB: bi, ValueB: b[bi]})
				i--
				j--
				continue
			}
		}

		if i > 0 {
			ai := i - 1
			if j > 0 {
				bi := j - 1
				cost = f(a[ai], b[bi])
			} else {
				cost = f(a[ai], nil)
			}
			if cost.DeleteA != Inhibit && matrix[i][j] == matrix[i-1][j]+cost.DeleteA {
				// A delete is preferred over an insert.
				ops = append(ops, Op{Kind: "delete", ValueA: a[ai]})
				i--
				continue
			}
		}

		if j > 0 {
			bi := j - 1
			if i > 0 {
				ai := i - 1
				cost = f(a[ai], b[bi])
			} else {
				cost = f(nil, b[bi])
			}
			if cost.InsertB != Inhibit && matrix[i][j] == matrix[i][j-1]+cost.InsertB {
				// Insert is the last preference.
				ops = append(ops, Op{Kind: "insert", IdxB: bi, ValueB: b[bi]})
				j--
				continue
			}
		}
	}

	// Reverse operations to be in the correct order.
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	// Calculate IdxA so that it corresponds to the index of a for each operation as they would be applied.
	idxA := 0
	for k, op := range ops {
		ops[k].IdxA = idxA
		if op.Kind == "match" || op.Kind == "replace" || op.Kind == "insert" {
			// Match, replace and insert advance idxA.
			idxA++
		} else if op.Kind == "delete" {
			// Deletion does not advance idxA.
			continue
		}
	}

	return int64(matrix[al][bl]), ops
}

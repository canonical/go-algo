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

package listdist

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

func Distance(a, b []any, f CostFunc, cut int64) int64 {
	lst := make([]CostInt, len(b)+1)
	bl := 0
	for bi, br := range b {
		bl++
		cost := f(nil, br)
		if cost.InsertB == Inhibit || lst[bi] == Inhibit {
			lst[bi+1] = Inhibit
		} else {
			lst[bi+1] = lst[bi] + cost.InsertB
		}
	}
	lst = lst[:bl+1]
	for _, ar := range a {
		last := lst[0]
		cost := f(ar, nil)
		if cost.DeleteA == Inhibit || last == Inhibit {
			lst[0] = Inhibit
		} else {
			lst[0] = last + cost.DeleteA
		}
		stop := true
		i := 0
		for _, br := range b {
			i++
			cost := f(ar, br)
			min := CostInt(Inhibit)
			if ar == br {
				min = last
			} else if cost.SwapAB != Inhibit && last != Inhibit {
				min = last + cost.SwapAB
			}
			if cost.InsertB != Inhibit && lst[i-1] != Inhibit {
				if n := lst[i-1] + cost.InsertB; n < min {
					min = n
				}
			}
			if cost.DeleteA != Inhibit && lst[i] != Inhibit {
				if n := lst[i] + cost.DeleteA; n < min {
					min = n
				}
			}
			last, lst[i] = lst[i], min
			if min < CostInt(cut) {
				stop = false
			}
		}
		_ = stop
		if cut != 0 && stop {
			break
		}
	}
	return int64(lst[len(lst)-1])
}

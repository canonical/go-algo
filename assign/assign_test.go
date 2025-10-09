package assign_test

import (
	"fmt"
	"testing"

	"github.com/canonical/go-algo/assign"

	. "gopkg.in/check.v1"
)

func nodeName(n any) string {
	if n == nil {
		return "-"
	}
	switch n := n.(type) {
	case string:
		return n
	default:
		panic(fmt.Sprintf("unknown test node: %T", n))
	}
}

type namePair struct {
	a, b string
}

type uintCost uint32

func (u uintCost) Less(other assign.Cost) bool { return u < other.(uintCost) }
func (u uintCost) String() string              { return fmt.Sprint(uint32(u)) }

type costMap map[namePair]uintCost

var minCost = uintCost(0)
var maxCost = uintCost(1 << 31)

func addCost(a, b assign.Cost) assign.Cost { return a.(uintCost) + b.(uintCost) }
func subCost(a, b assign.Cost) assign.Cost { return a.(uintCost) - b.(uintCost) }

func costFunc(costs map[namePair]uintCost) (editCost func(source, target any) assign.Cost) {
	return func(source, target any) assign.Cost {
		cost, ok := costs[namePair{nodeName(source), nodeName(target)}]
		if !ok {
			if source == nil || target == nil {
				// Deletions and insertions are better than misleading edits.
				return maxCost - 1
			}
			return maxCost
		}
		return cost
	}
}

func deltaOptions(costs costMap) *assign.AssignOptions {
	return &assign.AssignOptions{
		NodeKey:  func(n any) any { return n },
		EditCost: costFunc(costs),
		AddCost:  addCost,
		SubCost:  subCost,
		MinCost:  minCost,
		MaxCost:  maxCost,
	}
}

func pairsCost(pairs []assign.Pair) costMap {
	var result = make(costMap)
	for _, pair := range pairs {
		result[namePair{nodeName(pair.Source), nodeName(pair.Target)}] = pair.Cost.(uintCost)
	}
	return result
}

func (*S) TestEmpty(c *C) {
	options := deltaOptions(nil)
	pairs := assign.Assign([]any{}, []any{}, options)
	c.Assert(pairs, HasLen, 0)
	pairs = assign.Assign(nil, nil, options)
	c.Assert(pairs, HasLen, 0)
}

func (*S) TestTable(c *C) {
	for _, test := range deltaTests {
		c.Logf("Summary: %s", test.summary)
		options := deltaOptions(test.costs)
		sourceNodes := test.source
		targetNodes := test.target
		pairs := assign.Assign(sourceNodes, targetNodes, options)
		c.Assert(pairsCost(pairs), DeepEquals, test.result)
	}
}

type deltaTest struct {
	summary string
	costs   costMap
	source  []any
	target  []any
	result  costMap
}

var deltaTests = []deltaTest{{
	summary: "Update with max cost becomes delete+insert",
	costs:   costMap{namePair{"a", "b"}: maxCost},
	source:  []any{"a"},
	target:  []any{"b"},
	result: costMap{
		namePair{"a", "-"}: maxCost,
		namePair{"-", "b"}: maxCost,
	},
}, {
	summary: "Same as before but with a non-max update cost",
	costs:   costMap{namePair{"a", "b"}: maxCost - 1},
	source:  []any{"a"},
	target:  []any{"b"},
	result: costMap{
		namePair{"a", "b"}: maxCost - 1,
	},
}, {
	summary: "Prefer overall cheaper updates",
	costs: costMap{
		namePair{"a", "d"}: 1,
		namePair{"b", "e"}: 2,
		namePair{"c", "f"}: 4,
		namePair{"a", "e"}: 1,
		namePair{"b", "f"}: 2,
		namePair{"c", "d"}: 3, // Overall cost difference of just 1.
	},
	source: []any{"a", "b", "c", "x"},
	target: []any{"d", "e", "f", "y"},
	result: costMap{
		namePair{"a", "e"}: 1,
		namePair{"b", "f"}: 2,
		namePair{"c", "d"}: 3,
		namePair{"x", "-"}: maxCost,
		namePair{"-", "y"}: maxCost,
	},
}}

func benchmarkDelta(n int, b *testing.B) {
	source := make([]any, n)
	target := make([]any, n)
	costs := make(costMap)

	for i := 0; i < n; i++ {
		si := fmt.Sprintf("s%d", i)
		ti := fmt.Sprintf("t%d", i)
		source[i] = si
		target[i] = ti
		// Cost for matching s_i with t_i is low.
		costs[namePair{si, ti}] = 1
	}

	// The cost for insertions and deletions is handled by the costFunc
	// which returns maxCost-1. This is high enough to discourage them
	// in this benchmark's setup.

	options := deltaOptions(costs)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		assign.Assign(source, target, options)
	}
}

func BenchmarkDelta10(b *testing.B) {
	benchmarkDelta(10, b)
}

func BenchmarkDelta100(b *testing.B) {
	benchmarkDelta(100, b)
}

func BenchmarkDelta1000(b *testing.B) {
	benchmarkDelta(1000, b)
}

func BenchmarkDelta(b *testing.B) {
	for _, n := range []int{10, 20, 50, 100, 200, 1000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			benchmarkDelta(n, b)
		})
	}
}

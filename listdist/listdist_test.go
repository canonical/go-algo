package listdist_test

import (
	. "gopkg.in/check.v1"

	"testing"

	"github.com/canonical/editdelta/listdist"
)

type distanceTest struct {
	a, b string
	f    listdist.CostFunc
	r    int64
	cut  int64
}

func uniqueCost(ar, br any) listdist.Cost {
	return listdist.Cost{SwapAB: 1, DeleteA: 3, InsertB: 5}
}

var distanceTests = []distanceTest{
	{f: uniqueCost, r: 0, a: "abc", b: "abc"},
	{f: uniqueCost, r: 1, a: "abc", b: "abd"},
	{f: uniqueCost, r: 1, a: "abc", b: "adc"},
	{f: uniqueCost, r: 1, a: "abc", b: "dbc"},
	{f: uniqueCost, r: 2, a: "abc", b: "add"},
	{f: uniqueCost, r: 2, a: "abc", b: "ddc"},
	{f: uniqueCost, r: 2, a: "abc", b: "dbd"},
	{f: uniqueCost, r: 3, a: "abc", b: "ddd"},
	{f: uniqueCost, r: 3, a: "abc", b: "ab"},
	{f: uniqueCost, r: 3, a: "abc", b: "bc"},
	{f: uniqueCost, r: 3, a: "abc", b: "ac"},
	{f: uniqueCost, r: 6, a: "abc", b: "a"},
	{f: uniqueCost, r: 6, a: "abc", b: "b"},
	{f: uniqueCost, r: 6, a: "abc", b: "c"},
	{f: uniqueCost, r: 9, a: "abc", b: ""},
	{f: uniqueCost, r: 5, a: "abc", b: "abcd"},
	{f: uniqueCost, r: 5, a: "abc", b: "dabc"},
	{f: uniqueCost, r: 10, a: "abc", b: "adbdc"},
	{f: uniqueCost, r: 10, a: "abc", b: "dabcd"},
	{f: uniqueCost, r: 40, a: "abc", b: "ddaddbddcdd"},
	{f: listdist.StandardCost, r: 3, a: "abcdefg", b: "axcdfgh"},
	{f: listdist.StandardCost, r: 2, cut: 2, a: "abcdef", b: "abc"},
	{f: listdist.StandardCost, r: 2, cut: 3, a: "abcdef", b: "abcd"},
}

func (s *S) TestDistance(c *C) {
	for _, test := range distanceTests {
		c.Logf("Test: %v", test)
		f := test.f
		if f == nil {
			f = listdist.StandardCost
		}
		alist := splitString(test.a)
		blist := splitString(test.b)
		r := listdist.Distance(alist, blist, f, test.cut)
		c.Assert(r, Equals, test.r)
	}
}

func splitString(s string) []any {
	r := make([]any, len(s))
	for i, c := range s {
		r[i] = string(c)
	}
	return r
}

func BenchmarkDistance(b *testing.B) {
	one := splitString("abdefghijklmnopqrstuvwxyz")
	two := splitString("a.d.f.h.j.l.n.p.r.t.v.x.z")
	for i := 0; i < b.N; i++ {
		listdist.Distance(one, two, listdist.StandardCost, 0)
	}
}

func BenchmarkDistanceCut(b *testing.B) {
	one := splitString("abdefghijklmnopqrstuvwxyz")
	two := splitString("a.d.f.h.j.l.n.p.r.t.v.x.z")
	for i := 0; i < b.N; i++ {
		listdist.Distance(one, two, listdist.StandardCost, 1)
	}
}

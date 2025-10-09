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

package assign

type Pair struct {
	Source any
	Target any
	Cost   Cost
}

type AssignOptions struct {
	NodeKey  func(node any) any
	EditCost func(source, target any) Cost
	AddCost  func(a, b Cost) Cost
	SubCost  func(a, b Cost) Cost

	// MinCost is the minimum possible cost for an edit.
	// Besides implementing the Cost interface, it must be comparable by identity (==).
	MinCost Cost

	// MaxCost is the maximum possible cost for an edit.
	// Besides implementing the Cost interface, it must be comparable by identity (==)
	MaxCost Cost
}

// Assign returns the minimum cost pairs assigning each provided source
// into one of the provided targets, disregarding order. Inserts and
// deletes are represented by pairing a source or target with nil.
// The cost for each assignment is determined by the provided options.
//
// This is an implementation of https://en.wikipedia.org/wiki/Hungarian_algorithm
// (and hopefully remains O(n^3)) which is one of the well known solutions for the
// assignment problem: https://en.wikipedia.org/wiki/Assignment_problem
func Assign(sources, targets []any, options *AssignOptions) []Pair {
	n := len(sources)
	m := len(targets)

	// The cost matrix is square, as required by optimalCost.
	size := n
	if m > n {
		size = m
	}

	costs := make([][]Cost, size)
	for i := 0; i < size; i++ {
		costs[i] = make([]Cost, size)
		for j := 0; j < size; j++ {
			costs[i][j] = options.MinCost
		}
	}

	// Cost of substitution (source[i] -> target[j]).
	// Substitutions at MaxCost are later translated to insertions and deletions instead.
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			costs[i][j] = options.EditCost(sources[i], targets[j])
		}
	}

	// If n > m, sources i >= m are matched with nil target nodes. This is a deletion.
	for i := 0; i < n; i++ {
		cost := options.EditCost(sources[i], nil)
		for j := m; j < size; j++ {
			costs[i][j] = cost
		}
	}

	// If m > n, targets j >= n are matched with nil source nodes. This is an insertion.
	for j := 0; j < m; j++ {
		cost := options.EditCost(nil, targets[j])
		for i := n; i < size; i++ {
			costs[i][j] = cost
		}
	}

	optimal := optimalCost(costs, options)

	var result []Pair
	for j := 0; j < size; j++ {
		i := optimal[j]
		cost := costs[i][j]
		switch {
		case i < n && j < m:
			if cost == options.MaxCost {
				// Remove + Insert
				result = append(result, Pair{Source: sources[i], Target: nil, Cost: cost})
				result = append(result, Pair{Source: nil, Target: targets[j], Cost: cost})
			} else {
				// Update
				result = append(result, Pair{Source: sources[i], Target: targets[j], Cost: cost})
			}
		case i < n && j >= m:
			// Remove
			result = append(result, Pair{Source: sources[i], Target: nil, Cost: cost})
		case i >= n && j < m:
			// Insert
			result = append(result, Pair{Source: nil, Target: targets[i], Cost: cost})
		}
	}

	return result
}

type Cost interface {
	Less(other Cost) bool
}

// optimalCost returns an array where result[j] = i means target node j is matched
// with source node i. The cost matrix must be square, and costs[i][j] is the cost
// of matching left node i with right node j.
func optimalCost(costs [][]Cost, options *AssignOptions) []int {

	// The augmented path search works by taking a partial match between source and
	// target nodes (targetSource), which is better from a cost perspective but not yet
	// complete, and finding the next best option with additional nodes. The iteration
	// works by taking an arbitrary unassigned source node and finding the best target
	// node to add to the path, which may already be assigned to a source node, which
	// will need a new best target node, and so on, until we find an unassigned target
	// node. This process creates a trail of target nodes (targetTrail) that are all
	// "flipped" at the end, to reflect these reassignments. The process then repeats
	// until we have a complete matching for all nodes at the best total cost.
	//
	// The process of finding this path is similar to Dijkstra's algorithm for finding
	// the shortest path in a graph, where we explore all possible edges from the current
	// source node and then choose the edge with the minimum slack to extend the path.

	// The algorithm uses n+1 sized slices and marker values at n to simplify the logic.
	n := len(costs)

	// sourceCost[i] and targetCost[j] are partial costs for source and target nodes.
	// They maintain the "dual feasibility": sourceCost[i] + targetCost[j] <= cost[i][j].
	// Edges where sourceCost[i] + targetCost[j] == cost[i][j] are considered "tight",
	// meaning there is no slack to be removed, and form the equality subgraph.
	sourceCost := make([]Cost, n+1)
	targetCost := make([]Cost, n+1)

	// targetSource[j] = i stores the source node i matched with target node j.
	// A value of n means target node j is unmatched.
	targetSource := make([]int, n+1)

	for i := 0; i <= n; i++ {
		sourceCost[i] = options.MinCost
		targetCost[i] = options.MinCost
		targetSource[i] = n
	}

	// minSlack[j] stores the minimum slack for target node j, where the slack
	// is the difference between cost[i][j] and the sum of the partial costs.
	minSlack := make([]Cost, n+1)

	// targetTrail[j] stores the previous target node in the alternating path for target node j.
	// It is used to flip the matches along the trail when an augmenting path is found.
	targetTrail := make([]int, n+1)

	// visitedTarget[j] marks target nodes that are already in the trail.
	visitedTarget := make([]bool, n+1)

	// Main loop: find a good target for each source node i.
	for i := 0; i < n; i++ {
		// Start search for an augmenting path starting at source node i.
		// We use a dummy target node 0 to simplify the algorithm.
		targetSource[n] = i
		currentTarget := n

		for j := 0; j <= n; j++ {
			minSlack[j] = options.MaxCost
			targetTrail[j] = n
			visitedTarget[j] = false
		}

		// The loop continues until an unmatched target is found, which then extends the path.
		for targetSource[currentTarget] != n {
			visitedTarget[currentTarget] = true
			currentSource := targetSource[currentTarget]
			delta := options.MaxCost
			nextTarget := 0

			// Find the edge with the minimum slack to an unvisited target node.
			for j := 0; j < n; j++ {
				if !visitedTarget[j] {
					cost := costs[currentSource][j]
					curSlack := options.SubCost(cost, sourceCost[currentSource])
					curSlack = options.SubCost(curSlack, targetCost[j])
					if curSlack.Less(minSlack[j]) {
						minSlack[j] = curSlack
						targetTrail[j] = currentTarget
					}
					if minSlack[j].Less(delta) {
						delta = minSlack[j]
						nextTarget = j
					}
				}
			}

			// Update partial costs using delta. This makes at least one new edge "tight"
			// (have zero slack), allowing the alternating path to be extended.
			for j := 0; j <= n; j++ {
				if visitedTarget[j] {
					// For visited nodes, update partial costs to maintain tightness
					// of edges in the alternating path.
					i := targetSource[j]
					sourceCost[i] = options.AddCost(sourceCost[i], delta)
					targetCost[j] = options.SubCost(targetCost[j], delta)
				} else {
					// For unvisited nodes, decrease their slack by delta.
					minSlack[j] = options.SubCost(minSlack[j], delta)
				}
			}

			// The next target node is any of the ones that just became tight.
			currentTarget = nextTarget
		}

		// An augmented path was found, so fix the mapping by flipping
		// the edges along this path. The logic is trivial, but without a
		// proper example it's hard to visualize.
		//
		// If this is the current matching:
		//
		//    Sources:  A   B   C   D
		//                  |   |
		//    Targets:  X   Y   W   Z
		//
		// It means targetSource is Y => B, W => C.
		//
		// Then, assume this optimal path is found:
		//
		//    Sources:  A   B   C   D
		//                \ | \ | \
		//    Targets:  X   Y   W   Z
		//
		// This would create a targetTrail of Z => W, W => Y.
		//
		// The logic below will "flip" the matching so it becomes:
		//
		//    Sources:  A   B   C   D
		//                \   \   \
		//    Targets:  X   Y   W   Z
		//
		// So targetSource ends up as Y => A, W => B, Z => D.
		//
		// Quiz: Where do we get A from, if it's not in the previous matching, or the trail?
		//
		// Answer: Remember the n+1 simplification? We injected n as a fake target,
		//         so I lied to you. We actually had n => A in targetSource.
		//
		for currentTarget != n {
			previousTarget := targetTrail[currentTarget]
			targetSource[currentTarget] = targetSource[previousTarget]
			currentTarget = previousTarget
		}
	}

	// result[j] = i means target node j is matched with source node i.
	return targetSource[:n]
}

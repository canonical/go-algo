package main

import (
	"fmt"
	"os"

	"github.com/canonical/go-algo/assign"
)

// Custom Cost type, mirroring the one used in edit_test.go for consistency.
type uintCost uint32

func (u uintCost) Less(other assign.Cost) bool { return u < other.(uintCost) }
func (u uintCost) String() string              { return fmt.Sprint(uint32(u)) }

var (
	minCost = uintCost(0)
	maxCost = uintCost(1 << 31) // A very large cost to represent "impossible" or highly undesirable edits.
)

type agent struct {
	id int
}

type task struct {
	id int
}

func editCost(source, target any) assign.Cost {
	// The Hungarian Algorithm works on a square cost matrix, so if the number of agents
	// and tasks differ, nil agents or tasks will be used to match the number of agents
	// and tasks. Use minCost for these cases to represent no actual cost and to not affect
	// the optimal assignment.
	if source == nil || target == nil {
		return minCost
	}

	svalue, sok := source.(agent)
	tvalue, tok := target.(task)
	if !sok || !tok {
		panic("unknown value type")
	}

	// Known costs for possible agent-task assignments.
	cost := map[int]map[int]uintCost{
		0: {0: 8, 1: 4, 2: 7},
		1: {0: 5, 1: 2, 2: 3},
		2: {0: 9, 1: 6, 2: 7},
		3: {0: 9, 1: 4, 2: 8},
	}
	return cost[svalue.id][tvalue.id]
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	sources := []any{agent{id: 0}, agent{id: 1}, agent{id: 2}, agent{id: 3}}
	targets := []any{task{id: 0}, task{id: 1}, task{id: 2}}

	options := &assign.AssignOptions{
		NodeKey: func(node any) any {
			// NodeKey is used to identify nodes. For agent and task, the id serves as a unique identifier.
			// While not directly used by the Hungarian algorithm for cost, it's a required field
			// and provides a logical identifier for the nodes.
			if agent, ok := node.(agent); ok {
				return agent.id
			}
			if task, ok := node.(task); ok {
				return task.id
			}
			return nil // Should not happen with valid agent/task inputs
		},
		EditCost: editCost,
		AddCost: func(a, b assign.Cost) assign.Cost {
			return a.(uintCost) + b.(uintCost)
		},
		SubCost: func(a, b assign.Cost) assign.Cost {
			return a.(uintCost) - b.(uintCost)
		},
		MinCost: minCost,
		MaxCost: maxCost,
	}

	pairs := assign.Assign(sources, targets, options)

	// Process and print the diff results in the requested format.
	for _, p := range pairs {
		svalue, sok := p.Source.(agent)
		tvalue, tok := p.Target.(task)

		if !sok || !tok {
			// Made-up assignment with nil agent/task and minCost to balance number of agents
			// and tasks as required by the Hungarian Algorithm, skip these.
			continue
		}

		fmt.Printf("Agent: %d => Task: %d (with cost: %d)\n", svalue.id, tvalue.id, editCost(svalue, tvalue))
	}
	return nil
}

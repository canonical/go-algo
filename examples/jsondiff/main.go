package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/canonical/editdelta/assign"
	"github.com/canonical/editdelta/listdist"
)

// Custom Cost type, mirroring the one used in edit_test.go for consistency.
type uintCost uint32

func (u uintCost) Less(other assign.Cost) bool { return u < other.(uintCost) }
func (u uintCost) String() string              { return fmt.Sprint(uint32(u)) }

var (
	minCost = uintCost(0)
	maxCost = uintCost(1 << 31) // A very large cost to represent "impossible" or highly undesirable edits.
)

type jsonValue struct {
	path string
	data any
}

func flatten(nodes []any, data any, currentPath string) []any {
	nodes = append(nodes, jsonValue{path: currentPath, data: data})
	switch data := data.(type) {
	case map[string]any:
		for k, subdata := range data {
			var newPath string
			if currentPath == "." {
				newPath = "." + k
			} else {
				newPath = currentPath + "." + k
			}
			nodes = flatten(nodes, subdata, newPath)
		}
	case []any:
		for i, subdata := range data {
			newPath := fmt.Sprintf("%s[%d]", currentPath, i)
			nodes = flatten(nodes, subdata, newPath)
		}
	}
	return nodes
}

func isScalar(data any) bool {
	switch reflect.TypeOf(data).Kind() {
	case reflect.Bool, reflect.Int, reflect.Float64, reflect.String:
		return true
	}
	return false
}

func editCost(source, target any) assign.Cost {
	if source == nil || target == nil {
		return maxCost
	}

	svalue, sok := source.(jsonValue)
	tvalue, tok := target.(jsonValue)
	if !sok || !tok {
		panic("unknown value type")
	}

	isSourceRoot := svalue.path == "."
	isTargetRoot := tvalue.path == "."

	if isSourceRoot && isTargetRoot {
		// Matching root-to-root has zero cost.
		return minCost
	}
	if isSourceRoot || isTargetRoot {
		// Matching a root object to anything other than the other root is disallowed.
		return maxCost
	}

	sdata := svalue.data
	tdata := tvalue.data

	stype := reflect.TypeOf(sdata)
	ttype := reflect.TypeOf(tdata)

	// Disallow conversions between scalars or different types.
	// If types are fundamentally different, it's an impossible direct transformation,
	// so return MaxCost, which will become a delete + insert.
	if stype != ttype {
		return maxCost
	}

	// Same type, now compare content based on type.
	switch sdata := sdata.(type) {
	case nil:
		return minCost
	case bool, int, float64, string:
		if reflect.DeepEqual(sdata, tdata) {
			return minCost
		}
		if svalue.path == tvalue.path {
			return uintCost(1)
		}
		return maxCost // Replace.
	case map[string]any:
		tmap := tdata.(map[string]any)
		matchingKeys := 0
		allKeys := make(map[string]struct{})

		// Collect all unique keys from both maps and count matching keys
		for k := range sdata {
			allKeys[k] = struct{}{}
			if _, ok := tmap[k]; ok {
				matchingKeys++
			}
		}
		for k := range tmap {
			allKeys[k] = struct{}{}
		}

		totalUniqueKeys := len(allKeys)
		if totalUniqueKeys == 0 {
			return minCost
		}
		return uintCost(totalUniqueKeys - matchingKeys)

	case []any:
		tdata := tdata.([]any)
		return uintCost(listdist.Distance(sdata, tdata, listdist.StandardCost, 0))

	default:
		return maxCost
	}
}

func formatValue(val any) string {
	b, err := json.Marshal(val)
	if err != nil {
		return fmt.Sprintf("<error marshalling: %v>", err)
	}
	return string(b)
}

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file1> <file2>\n", os.Args[0])
		os.Exit(1)
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	file1Path := flag.Arg(0)
	file2Path := flag.Arg(1)

	data1, err := ioutil.ReadFile(file1Path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %v", file1Path, err)
	}

	data2, err := ioutil.ReadFile(file2Path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %v", file2Path, err)
	}

	var json1, json2 any
	if err := json.Unmarshal(data1, &json1); err != nil {
		return fmt.Errorf("cannot unmarshal %s: %v", file1Path, err)
	}
	if err := json.Unmarshal(data2, &json2); err != nil {
		return fmt.Errorf("cannot unmarshal %s: %v", file2Path, err)
	}

	sources := flatten(nil, json1, ".")
	targets := flatten(nil, json2, ".")

	options := &assign.AssignOptions{
		NodeKey: func(node any) any {
			// NodeKey is used to identify nodes. For JSONNode, the path serves as a unique identifier.
			// While not directly used by the Hungarian algorithm for cost, it's a required field
			// and provides a logical identifier for the nodes.
			if jn, ok := node.(jsonValue); ok {
				return jn.path
			}
			return nil // Should not happen with valid JSONNode inputs
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
		svalue, sok := p.Source.(jsonValue)
		tvalue, tok := p.Target.(jsonValue)

		switch {
		case sok && !tok:
			fmt.Printf("Drop: old%s\n", svalue.path)
		case !sok && tok:
			if tvalue.path == "." && (reflect.DeepEqual(tvalue.data, map[string]any{}) || reflect.DeepEqual(tvalue.data, []any{})) {
				continue
			}
			fmt.Printf(" Add: new%s = %s\n", tvalue.path, formatValue(tvalue.data))
		case sok && tok:
			if svalue.path == tvalue.path {
				if svalue.path == "." && tvalue.path == "." {
					continue
				}
				if isScalar(svalue.data) && !reflect.DeepEqual(svalue.data, tvalue.data) {
					fmt.Printf(" Set: new%s = %s\n", svalue.path, formatValue(tvalue.data))
				}
			} else {
				if reflect.DeepEqual(svalue.data, tvalue.data) {
					fmt.Printf("Move: old%s => new%s\n", svalue.path, tvalue.path)
				} else if isScalar(svalue.data) && isScalar(tvalue.data) {
					fmt.Printf("Move: old%s => new%s = %s\n", svalue.path, tvalue.path, formatValue(tvalue.data))
				} else {
					fmt.Printf("Move: old%s => new%s\n", svalue.path, tvalue.path)
				}
			}
		}
	}
	return nil
}

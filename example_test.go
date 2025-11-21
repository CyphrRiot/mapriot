package mapriot

import (
	"fmt"
	"sort"
	"strings"
)

// ExampleMapReduce_wordCount demonstrates a simple word-count using MapRiot.
func ExampleMapReduce_wordCount() {
	lines := []string{
		"to be or not to be",
		"that is the question",
		"to be",
	}

	mapFn := func(s string) []KV[string, int] {
		fields := strings.Fields(strings.ToLower(s))
		out := make([]KV[string, int], 0, len(fields))
		for _, w := range fields {
			out = append(out, KV[string, int]{K: w, V: 1})
		}
		return out
	}

	reduceFn := func(word string, counts []int) int {
		sum := 0
		for _, c := range counts {
			sum += c
		}
		return sum
	}

	result := MapReduce(lines, mapFn, reduceFn, 0)

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s %d\n", k, result[k])
	}
	// Output:
	// be 3
	// is 1
	// not 1
	// or 1
	// question 1
	// that 1
	// the 1
	// to 3
}

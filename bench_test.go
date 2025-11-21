package mapriot

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// naiveWordCountBaseline is a single-threaded reference implementation
// that mirrors the MapReduce logic without goroutines.
func naiveWordCountBaseline(lines []string) map[string]int {
	groups := make(map[string][]int)
	for _, line := range lines {
		fields := strings.Fields(strings.ToLower(line))
		for _, w := range fields {
			groups[w] = append(groups[w], 1)
		}
	}
	out := make(map[string]int, len(groups))
	for k, vs := range groups {
		s := 0
		for _, v := range vs {
			s += v
		}
		out[k] = s
	}
	return out
}

func makeRandomLines(n int, words []string) []string {
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		// Random length between 5 and 12 words
		l := 5 + rand.IntN(8)
		b := new(strings.Builder)
		for j := 0; j < l; j++ {
			if j > 0 {
				b.WriteByte(' ')
			}
			w := words[rand.IntN(len(words))]
			b.WriteString(w)
		}
		lines[i] = b.String()
	}
	return lines
}

func BenchmarkBaseline_WordCount_Small(b *testing.B) {
	lines := []string{
		"to be or not to be",
		"that is the question",
		"to be",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = naiveWordCountBaseline(lines)
	}
}

func BenchmarkMapRiot_WordCount_Small(b *testing.B) {
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
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = MapReduce(lines, mapFn, reduceFn, 0)
	}
}

func BenchmarkBaseline_WordCount_Large(b *testing.B) {
	words := []string{"to", "be", "or", "not", "that", "is", "the", "question", "lorem", "ipsum", "dolor", "sit", "amet"}
	lines := makeRandomLines(10000, words)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = naiveWordCountBaseline(lines)
	}
}

func BenchmarkMapRiot_WordCount_Large(b *testing.B) {
	words := []string{"to", "be", "or", "not", "that", "is", "the", "question", "lorem", "ipsum", "dolor", "sit", "amet"}
	lines := makeRandomLines(10000, words)
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
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = MapReduce(lines, mapFn, reduceFn, 0)
	}
}

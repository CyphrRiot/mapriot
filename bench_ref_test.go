package mapriot

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	ref "github.com/Jitesh117/mapReduceGo/mapreduce"
)

// silenceStdout runs fn while redirecting stdout to io.Discard.
func silenceStdout(fn func()) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	log.SetOutput(io.Discard)
	ch := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, r)
		close(ch)
	}()
	fn()
	_ = w.Close()
	<-ch
	os.Stdout = old
}

// wire the reference API into a comparable bench (word-count)
func BenchmarkReference_WordCount_Large(b *testing.B) {
	words := []string{"to", "be", "or", "not", "that", "is", "the", "question", "lorem", "ipsum", "dolor", "sit", "amet"}
	lines := makeRandomLines(10000, words)
	mapF := func(docID string, contents string) []ref.KeyValue {
		fields := strings.Fields(strings.ToLower(contents))
		out := make([]ref.KeyValue, 0, len(fields))
		for _, w := range fields {
			out = append(out, ref.KeyValue{Key: w, Value: "1"})
		}
		return out
	}
	reduceF := func(key string, values []string) string { return "" } // count only, ignore output
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		silenceStdout(func() {
			m := ref.NewMaster(lines, 3)
			var wg sync.WaitGroup
			wg.Add(4)
			for w := 0; w < 4; w++ {
				nw := ref.NewWorker(w, m, mapF, reduceF)
				go nw.Run(&wg)
			}
			wg.Wait()
		})
	}
}

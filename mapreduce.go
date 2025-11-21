package mapriot

import (
	"runtime"
	"sync"
)

// KV is a key/value pair emitted by map functions.
// It intentionally mirrors the simplicity found in Lisp-based MapReduce examples.
//
// Example:
//   out = append(out, KV[string,int]{K: word, V: 1})
//
// Grouping and reduction are handled by MapReduce.
type KV[K comparable, V any] struct {
	K K
	V V
}

// MapReduce applies mapFn to each item, groups values by key, then reduces each group with reduceFn.
//
// Design notes:
// - workers <= 0 defaults to runtime.GOMAXPROCS(0)
// - Each mapper goroutine uses a local combiner (map[K][]U) to minimize contention and allocations.
// - One channel for jobs, one for partials; a straightforward fan-in before reduction keeps it clear and fast.
// - The reduce stage is sequential by default (keep it simple and predictable). If parallel reduction is ever
//   needed, a similar worker-pool over grouped keys can be added without complicating the core.
func MapReduce[T any, K comparable, U any, R any](
	items []T,
	mapFn func(T) []KV[K, U],
	reduceFn func(K, []U) R,
	workers int,
) map[K]R {
	if len(items) == 0 {
		return make(map[K]R)
	}
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	// Fast path: sequential execution without goroutines or channels
	if workers <= 1 {
		grouped := make(map[K][]U)
		for _, it := range items {
			for _, kv := range mapFn(it) {
				grouped[kv.K] = append(grouped[kv.K], kv.V)
			}
		}
		out := make(map[K]R, len(grouped))
		for k, vs := range grouped {
			out[k] = reduceFn(k, vs)
		}
		return out
	}

	jobs := make(chan T)
	partials := make(chan map[K][]U, workers)

	var wg sync.WaitGroup
	wg.Add(workers)

	// Map workers with local combiners
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			local := make(map[K][]U)
			for it := range jobs {
				for _, kv := range mapFn(it) {
					local[kv.K] = append(local[kv.K], kv.V)
				}
			}
			partials <- local
		}()
	}

	// Feed items
	go func() {
		for _, it := range items {
			jobs <- it
		}
		close(jobs)
	}()

	// Close partials when all workers are done
	go func() {
		wg.Wait()
		close(partials)
	}()

	// Merge partial maps
	grouped := make(map[K][]U)
	for part := range partials {
		for k, vs := range part {
			grouped[k] = append(grouped[k], vs...)
		}
	}

	// Reduce (sequential)
	out := make(map[K]R, len(grouped))
	for k, vs := range grouped {
		out[k] = reduceFn(k, vs)
	}
	return out
}

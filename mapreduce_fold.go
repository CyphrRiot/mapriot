package mapriot

import (
	"runtime"
	"sync"
)

// MapReduceFold applies mapFn to each item and accumulates per-key values directly.
// It avoids materializing []U per key by folding values into an accumulator R.
//
// - mapFn:   produces key/value pairs for an input item
// - zeroFn:  returns the identity accumulator for a given key (e.g., 0 for sums)
// - foldFn:  folds a single mapped value into the accumulator (acc <- f(acc, v))
// - mergeFn: merges two accumulators (acc <- g(accA, accB)); should be associative
// - workers: <= 0 defaults to runtime.GOMAXPROCS(0)
//
// The algorithm:
//   1) Each worker maintains a local map[K]R accumulator.
//   2) For each item, it maps then folds values into its local accumulators.
//   3) Workers emit their local maps; the main goroutine merges accumulators by key.
//
// This mirrors Lisp's reduce semantics more closely and typically allocates less than
// a group-then-reduce approach, especially for numeric aggregations (counts, sums, etc.).
func MapReduceFold[T any, K comparable, U any, R any](
	items []T,
	mapFn func(T) []KV[K, U],
	zeroFn func(K) R,
	foldFn func(K, R, U) R,
	mergeFn func(K, R, R) R,
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
		out := make(map[K]R)
		for _, it := range items {
			pairs := mapFn(it)
			for i := range pairs {
				k := pairs[i].K
				v := pairs[i].V
				acc, ok := out[k]
				if !ok {
					acc = zeroFn(k)
				}
				out[k] = foldFn(k, acc, v)
			}
		}
		return out
	}

	// Parallel path: statically partition input across workers (no jobs channel)
	if workers > len(items) {
		workers = len(items)
	}
	partials := make(chan map[K]R, workers)

	var wg sync.WaitGroup
	wg.Add(workers)

	chunk := (len(items) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		start := w * chunk
		end := start + chunk
		if start >= len(items) {
			wg.Done()
			break
		}
		if end > len(items) {
			end = len(items)
		}
		segment := items[start:end]
		go func(seg []T) {
			defer wg.Done()
			local := make(map[K]R)
			for _, it := range seg {
				pairs := mapFn(it)
				for i := range pairs {
					k := pairs[i].K
					v := pairs[i].V
					acc, ok := local[k]
					if !ok {
						acc = zeroFn(k)
					}
					local[k] = foldFn(k, acc, v)
				}
			}
			partials <- local
		}(segment)
	}

	go func() {
		wg.Wait()
		close(partials)
	}()

	out := make(map[K]R)
	for part := range partials {
		for k, r := range part {
			if cur, ok := out[k]; ok {
				out[k] = mergeFn(k, cur, r)
			} else {
				out[k] = r
			}
		}
	}
	return out
}

# MapRiot: a tiny, fast MapReduce for Go (with Lisp spirit)

MapRiot keeps MapReduce to its essence: map over items, group by key, reduce each group. No framework bloat, no over‑abstracted interfaces. It mirrors how Lisp expresses MapReduce: higher‑order functions over simple data structures.

- Minimal API: pure map and reduce functions; small `KV` type.
- Efficient by default: local combiners per worker, minimal allocations, low contention.
- Concurrency that stays out of your way: one jobs channel, one partials channel.
- Practical performance: intended to be at least as fast as popular community examples, while being clearer and smaller.


## Status

Early alpha. API is small and likely to stay that way. Performance baselines and comparisons will be added in `bench_test.go`.


## Design goals

- Lisp‑like clarity: map → group → reduce.
- Pure functions in, pure data out.
- Keep concurrency off the hot path where it hurts clarity.
- Use local combiners to reduce fan‑in costs.
- Avoid reflection and interface sprawl; use generics for type safety.


## Common Lisp reference example

The following Common Lisp snippet shows the essence of MapReduce for word count. It uses a hash table to group values (the “shuffle” step), then reduces each group. This is the spirit MapRiot follows.

Note: uses `split-sequence` Quicklisp library for tokenization.

```lisp
;;;; word-count.lisp
;;;; (ql:quickload :split-sequence)

(defun word-count (lines)
  (let ((groups (make-hash-table :test 'equal)))
    ;; Map + local group (combiner)
    (dolist (line lines)
      (dolist (w (remove "" (split-sequence:split-sequence-if #'char-whitespace-p (string-downcase line))))
        (push 1 (gethash w groups '()))))
    ;; Reduce per key
    (let ((result (make-hash-table :test 'equal)))
      (maphash (lambda (k vs)
                 (setf (gethash k result)
                       (reduce #'+ vs)))
               groups)
      result)))

;; Example:
;; (word-count '("To be or not to be" "That is the question" "to be"))
```


## Go API

```go
package mapriot

// KV pairs used by the map function.
type KV[K comparable, V any] struct {
    K K
    V V
}

// MapReduce applies mapFn to each item, groups by key, then reduces each group with reduceFn.
// workers <= 0 defaults to GOMAXPROCS.
func MapReduce[T any, K comparable, U any, R any](
    items []T,
    mapFn func(T) []KV[K, U],
    reduceFn func(K, []U) R,
    workers int,
) map[K]R
```


## Go usage example (word count)

```go
package main

import (
    "fmt"
    "strings"

    "github.com/CyphrRiot/mapriot"
)

func main() {
    lines := []string{
        "to be or not to be",
        "that is the question",
        "to be",
    }

    mapFn := func(s string) []mapriot.KV[string, int] {
        fields := strings.Fields(strings.ToLower(s))
        out := make([]mapriot.KV[string, int], 0, len(fields))
        for _, w := range fields {
            out = append(out, mapriot.KV[string, int]{K: w, V: 1})
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

    result := mapriot.MapReduce(lines, mapFn, reduceFn, 0) // 0 => GOMAXPROCS
    fmt.Println(result)
}
```


## Performance and benchmarks

Goals:
- Be at least as fast as (and ideally faster than) common community examples, including the reference implementation discussed in <https://jitesh117.github.io/blog/implementing-mapreduce-in-golang/>.
- Maintain readability and a tiny surface area.

Approach:
- Local combiners in mapper goroutines to reduce shared contention.
- Pre‑allocate slices where possible; reuse in hot paths.
- Minimal synchronization: single producer channel for jobs, single channel for partial maps.

Benchmarks (to be added in `bench_test.go`):
- Naive single‑thread baseline.
- MapRiot implementation.
- Optional: comparison against the referenced repo’s implementation (run side‑by‑side on identical data).

Run benches:

```bash
go test -bench . -benchmem ./...
```


## Roadmap
- [ ] Initial implementation (`mapreduce.go`)
- [ ] Word‑count example and tests
- [ ] Benchmarks and comparison harness
- [ ] Optional: context cancellation and parallel reduce (only if needed)


## Module path

The intended module path is:

```
github.com/CyphrRiot/mapriot
```

We’ll initialize `go.mod` accordingly.


## License
TBD (MIT/Apache‑2.0/BSD‑3‑Clause). Please set your preference.

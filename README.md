# MapRiot


[![version](https://img.shields.io/badge/version-v0.1-blue.svg)](https://github.com/CyphrRiot/mapriot/releases/tag/v0.1) [![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)

Tiny, fast MapReduce for Go — inspired by Lisp.

MapRiot keeps MapReduce to its essence: map items, group by key, reduce each group. It mirrors the simplicity found in Lisp: higher‑order functions over simple data structures.

- Minimal API with a small `KV` type
- Efficient by default: local combiners, low contention
- Clear concurrency with small worker pools


## Design goals

- Lisp‑like clarity: map → group → reduce or map → fold → merge
- Pure functions in, pure data out
- Avoid reflection and interface sprawl; use generics


## Benchmarks (quick look)

Large input (10k lines):

| Implementation                 |   Time/op |  Mem/op | Allocs/op |
|--------------------------------|----------:|--------:|----------:|
| Reference (Jitesh117)          | 378.31ms  | 16.90MB |    134,524|
| MapRiot (group → reduce)       | 24.66ms   |  7.58MB |     22,034|
| MapRiotFold (fold → merge)     | 6.60ms    |  3.47MB |     20,223|

Small input (tiny workload):

| Implementation         |  Time/op |
|------------------------|---------:|
| Baseline single-thread |   ~2.59µs|
| MapRiot                |  ~25.10µs|
| MapRiotFold            |  ~10.90µs|

Notes:
- Measured on Intel(R) Core(TM) Ultra 7 258V, Go 1.25.3, Linux
- Reproduce: `go test -bench . -benchmem -run ^$`
- See “Compare with reference” for local setup details


Word count implemented with grouping and then reduction. This is the spirit MapRiot follows.

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

// MapReduce: map -> group -> reduce
func MapReduce[T any, K comparable, U any, R any](
    items []T,
    mapFn func(T) []KV[K, U],
    reduceFn func(K, []U) R,
    workers int,
) map[K]R

// MapReduceFold: map -> fold accumulators -> merge accumulators
func MapReduceFold[T any, K comparable, U any, R any](
    items []T,
    mapFn func(T) []KV[K, U],
    zeroFn func(K) R,
    foldFn func(K, R, U) R,
    mergeFn func(K, R, R) R,
    workers int,
) map[K]R
```


## Usage: MapReduce (word count)

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
        for _, c := range counts { sum += c }
        return sum
    }

    result := mapriot.MapReduce(lines, mapFn, reduceFn, 0) // 0 => GOMAXPROCS
    fmt.Println(result)
}
```


## Usage: MapReduceFold (word count)

```go
mapFn := func(s string) []mapriot.KV[string, int] {
    fields := strings.Fields(strings.ToLower(s))
    out := make([]mapriot.KV[string, int], 0, len(fields))
    for _, w := range fields {
        out = append(out, mapriot.KV[string, int]{K: w, V: 1})
    }
    return out
}
zeroFn  := func(string) int { return 0 }
foldFn  := func(_ string, acc, v int) int { return acc + v }
mergeFn := func(_ string, a, b int) int { return a + b }

result := mapriot.MapReduceFold(lines, mapFn, zeroFn, foldFn, mergeFn, 0)
```


## Workers and execution

- workers <= 0: defaults to runtime.GOMAXPROCS(0)
- workers <= 1: sequential fast‑path (no goroutines/channels)
- MapReduceFold uses static input partitioning to avoid a jobs channel
- Small inputs: set workers to 1 for lowest overhead
- Large numeric aggregations (counts/sums): prefer MapReduceFold for fewer allocations


## Performance and benchmarks

Benchmarks live in `bench_test.go`. Run:

```bash
go test -bench . -benchmem ./...
```

Notes:
- MapReduceFold removes []U materialization and typically allocates less
- The sequential fast‑path cuts overhead for small workloads


## Module path

```
github.com/CyphrRiot/mapriot
```


## License

MIT. See LICENSE for details.

## Compare with reference (Jitesh117)

This project includes an apples-to-apples benchmark against github.com/Jitesh117/mapReduceGo.

Local setup:
- Clone the reference repo beside this project:
  ```bash
  git clone https://github.com/Jitesh117/mapReduceGo.git ~/Code/mapReduceGo
  ```
- go.mod contains a local replace to point to that checkout:
  ```
  replace github.com/Jitesh117/mapReduceGo => /home/grendel/Code/mapReduceGo
  ```

Run only the reference bench:
```bash
go test -bench BenchmarkReference -benchmem -run ^$
```

Run all benches:
```bash
go test -bench . -benchmem -run ^$
```

Notes:
- The reference workers print to stdout; the benchmark silences stdout during measurement.
- Hardware, Go version, and dataset size affect results; compare relative numbers on the same machine.


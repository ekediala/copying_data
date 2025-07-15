# From ReadAll to CopyBuffer: A Go Developer’s Guide to Efficient Data Copying

A practical, evidence-based guide to choosing between `io.ReadAll`, `io.Copy`, and `io.CopyBuffer` for file and stream operations in Go.

---

## Introduction

When working with files, network streams, or other I/O in Go, developers often face a choice:

Should you load all data into memory, or process it as a stream?

This article presents real benchmark and memory profiling results for Go’s core data copying strategies, clarifies their tradeoffs, and offers clear guidance for real-world applications.

---

## Approaches Compared

### 1. `io.ReadAll`: Load Everything Into Memory

```go
func readBodyIoReadAll(r io.Reader, w io.Writer) error {
    b, err := io.ReadAll(r)
    if err != nil {
        return fmt.Errorf("reading data: %w", err)
    }
    if _, err := w.Write(b); err != nil {
        return fmt.Errorf("writing data: %w", err)
    }
    return nil
}
```
- Reads the entire input into memory before writing.
- Simple and idiomatic when you need the whole payload at once.

---

### 2. `io.Copy`: Stream Data in Chunks

```go
func readBodyBuffered(r io.Reader, w io.Writer) error {
    _, err := io.Copy(w, r)
    if err != nil {
        return fmt.Errorf("reading data: %w", err)
    }
    return nil
}
```
- Streams data using a fixed-size buffer (default: 32KB).
- Memory usage is constant, regardless of file size.

---

### 3. `io.CopyBuffer` + `sync.Pool`: Optimized Streaming

```go
var bufferPool = sync.Pool{
    New: func() any {
        return make([]byte, 32*1024)
    },
}

func readBodyBufferedPool(r io.Reader, w io.Writer) error {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    _, err := io.CopyBuffer(w, r, buf)
    if err != nil {
        return fmt.Errorf("reading data: %w", err)
    }
    return nil
}
```
- Reuses buffers across operations.
- Reduces allocations in high-throughput or concurrent scenarios.

---

## Benchmark Results

Benchmarks were run using Go’s `testing` package on an Apple M1 Pro (darwin/arm64):

```
goos: darwin
goarch: arm64
cpu: Apple M1 Pro
```

| Benchmark                                 | ns/op | B/op   | allocs/op |
|--------------------------------------------|-------|--------|-----------|
| IoReadAll_SmartRaise_Dash.jpg              | 1081  | 519    | 1         |
| io.Copy_SmartRaise_Dash.jpg                | 2908  | 32784  | 3         |
| io.CopyBuffer_SmartRaise_Dash.jpg          | 2975  | 32784  | 3         |
| IoReadAll_README.md                        | 1082  | 512    | 1         |
| io.Copy_README.md                          | 2915  | 32784  | 3         |
| io.CopyBuffer_README.md                    | 2870  | 32784  | 3         |

**Key Takeaways:**
- `io.ReadAll` is extremely fast and allocates very little memory for small files.
- `io.Copy` and `io.CopyBuffer` are slightly slower and always allocate a 32KB buffer, regardless of file size.
- All approaches have minimal allocations, but streaming methods always allocate at least the buffer.

---

## Memory Profiling Results

Memory profiling with `pprof` (using a 1.4mb file) showed:

- `io.ReadAll` allocates memory proportional to the file size.
- `io.Copy` and `io.CopyBuffer` (file-to-file) may show zero Go-level allocations due to OS-level optimizations (e.g., `sendfile`).
- When copying to a buffer (not a file), allocations are visible and reflect buffer growth.

---

## When to Use Each Approach

### Use `io.ReadAll` when:
- Your logic requires the entire payload in memory (e.g., validation, transformation, scanning, or random access).
- The data fits comfortably in memory (small/medium files or payloads).

### Use streaming (`io.Copy`, `io.CopyBuffer`) when:
- You can process data incrementally and do not need the whole payload at once.
- You are saving uploads directly to disk, forwarding to another service, or building pipelines.
- Memory efficiency is critical, especially for large files or high concurrency.

**Summary:**
Choose `io.ReadAll` for scenarios where full in-memory access is required, and prefer streaming for large data or when memory efficiency is critical and whole-data access is unnecessary.

---

## Benchmarking vs. Profiling: What’s the Difference?

- **Benchmarking** (via `go test -bench`) measures per-operation speed and allocations in isolation.
- **Memory profiling** (via `pprof`) shows real-world, end-to-end memory usage, including effects of OS-level optimizations and actual file I/O.
- Both are essential: benchmarks reveal micro-level costs, while profiling shows whole-program behavior.

---

## Practical Guidance

1. **Profile and benchmark with real workloads:** Synthetic tests can be misleading; always measure with realistic scenarios.
2. **`io.ReadAll` is efficient for small/medium data:** It’s fast and allocates only what’s needed.
3. **Streaming (`io.Copy`, `io.CopyBuffer`) is best for large data:** Memory usage is constant and independent of file size.
4. **OS-level optimizations matter:** File-to-file copying may use zero Go memory due to system calls like `sendfile`.
5. **Buffer reuse (`sync.Pool`) helps at scale:** For high-throughput or concurrent workloads, pooling buffers can further reduce allocations.

---

*Analysis conducted with Go 1.24.2 on macOS (Apple M1 Pro). Results may vary across different operating systems and Go versions. Source code and profiles available in this repository.*

---

## Complete Memory Profile Data

For those interested in the raw memory profiling results, here are the `pprof` outputs used in this analysis:

### io.ReadAll Profile (readall.prof)
```
File: bin
Type: inuse_space
Time: 2025-07-15 21:53:20 WAT
Showing nodes accounting for 2468.87kB, 100% of 2468.87kB total
      flat  flat%   sum%        cum   cum%
    1026kB 41.56% 41.56%     1026kB 41.56%  runtime.allocm
  930.82kB 37.70% 79.26%   930.82kB 37.70%  io.ReadAll
  512.05kB 20.74%   100%  1442.87kB 58.44%  runtime.main
         0     0%   100%   930.82kB 37.70%  main.main
         0     0%   100%   930.82kB 37.70%  main.readBodyIoReadAll

Function-level allocation:
ROUTINE ======================== main.readBodyIoReadAll
         0   930.82kB (flat, cum) 37.70% of Total
         .   930.82kB     20:   b, err := io.ReadAll(r)
```

### io.Copy Profile (copy.prof)
```
File: bin
Type: inuse_space
Time: 2025-07-15 21:53:25 WAT
Showing nodes accounting for 1546.28kB, 100% of 1546.28kB total
      flat  flat%   sum%        cum   cum%
 1033.28kB 66.82% 66.82%  1033.28kB 66.82%  runtime.procresize
     513kB 33.18%   100%      513kB 33.18%  runtime.allocm

No application-level allocations visible - OS optimization in effect
```

### sync.Pool Profile (syncpool.prof)
```
File: bin
Type: inuse_space
Time: 2025-07-15 21:53:30 WAT
Showing nodes accounting for 513kB, 100% of 513kB total
      flat  flat%   sum%        cum   cum%
     513kB   100%   100%      513kB   100%  runtime.allocm

Minimal runtime overhead only - perfect buffer reuse
```

*Note: Only the three profiles above were actually generated from the test code. All analysis is based on these real measurements.*

# When io.ReadAll Becomes a Performance Bottleneck in Go

*A performance comparison of data copying methods for large or unknown-sized data streams*

## Introduction

When working with HTTP responses, file I/O, or data streams in Go, developers often reach for `io.ReadAll()` as the go-to solution for reading data into memory. It's convenient, it's in the standard library, most API documentations use it in examples, and it "just works." For most applications processing small, predictable data sizes, this is perfectly fine.

However, when dealing with large or unknown-sized data streams—such as file uploads, API responses with variable payload sizes, or data processing pipelines—the performance characteristics of `io.ReadAll` can become problematic.

In this article, we'll explore different approaches to reading data in Go and examine when the convenience of `io.ReadAll` comes at a significant performance cost.

## The Setup: Two Approaches to Reading Data

Let's examine two common patterns for reading data from an `io.Reader`:

### Approach 1: The "Obvious" Way with io.ReadAll

```go
func readBodyIoReadAll(r io.Reader) ([]byte, error) {
    return io.ReadAll(r)
}
```

Simple, clean, and straightforward. This is what most developers would naturally reach for. All API documentations I have seen use this in their example.

### Approach 2: The Buffered Approach

```go
func readBodyBuffered(r io.Reader) ([]byte, error) {
    var buf bytes.Buffer
    _, err := io.Copy(&buf, r)
    if err != nil {
        return nil, fmt.Errorf("reading data: %w", err)
    }
    return buf.Bytes(), nil
}
```

This approach uses `bytes.Buffer` with `io.Copy`. It's slightly more verbose but, as we'll see, there's a good reason to prefer it.

**Optimization Note:** The buffered approach also allows us to optimize things further using `io.CopyBuffer`, which allows us to pass a preallocated buffer of our desired size (which can be from a `sync.Pool`) and as such eliminate the cost of continuously creating new slices only to discard them after copying the data. As always there are no free lunches and as such we must benchmark this to be certain there are practical benefits:

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 64*1024) // 64KB buffer
    },
}

func readBodyBufferedOptimized(r io.Reader) ([]byte, error) {
    var buf bytes.Buffer
    copyBuf := bufferPool.Get().([]byte)
    defer bufferPool.Put(copyBuf)

    _, err := io.CopyBuffer(&buf, r, copyBuf)
    if err != nil {
        return nil, fmt.Errorf("reading data: %w", err)
    }
    return buf.Bytes(), nil
}
```

This optimization is particularly valuable in high-throughput scenarios where the same operation is performed repeatedly.

## The Benchmark: Putting Theory to the Test

I created a comprehensive benchmark to test both approaches across different data sizes:

```go
func BenchmarkReadMethods(b *testing.B) {
    sizes := []int{1024, 10000, 100000, 1000000}

    for _, size := range sizes {
        data := make([]byte, size)

        b.Run(fmt.Sprintf("IoReadAll_%d", size), func(b *testing.B) {
            b.ReportAllocs()
            for i := 0; i < b.N; i++ {
                r := bytes.NewReader(data)
                _, err := readBodyIoReadAll(r)
                if err != nil {
                    b.Fatal(err)
                }
            }
        })

        b.Run(fmt.Sprintf("Buffered_%d", size), func(b *testing.B) {
            b.ReportAllocs()
            for i := 0; i < b.N; i++ {
                r := bytes.NewReader(data)
                _, err := readBodyBuffered(r)
                if err != nil {
                    b.Fatal(err)
                }
            }
        })
    }
}
```

## The Results: A Performance Story

Here are the benchmark results on an Apple M1 Pro:

```
BenchmarkReadMethods/IoReadAll_1024-10         3337831    355.7 ns/op    2864 B/op    4 allocs/op
BenchmarkReadMethods/Buffered_1024-10          6806582    176.2 ns/op    1120 B/op    3 allocs/op
BenchmarkReadMethods/IoReadAll_10000-10         265746   4556 ns/op    46128 B/op   11 allocs/op
BenchmarkReadMethods/Buffered_10000-10         1216825    983.6 ns/op   10336 B/op    3 allocs/op
BenchmarkReadMethods/IoReadAll_100000-10        27645  44683 ns/op   514363 B/op   19 allocs/op
BenchmarkReadMethods/Buffered_100000-10        144283   8272 ns/op   106594 B/op    3 allocs/op
BenchmarkReadMethods/IoReadAll_1000000-10        2845 414515 ns/op  5241270 B/op   32 allocs/op
BenchmarkReadMethods/Buffered_1000000-10        15804  76015 ns/op  1007733 B/op    3 allocs/op
```

## Breaking Down the Numbers

### Performance Gap Widens with Data Size

- **1KB**: Buffered approach is ~2x faster
- **10KB**: Buffered approach is ~4.6x faster
- **100KB**: Buffered approach is ~5.4x faster
- **1MB**: Buffered approach is ~5.5x faster

### Memory Usage: Scaling Becomes Problematic

The memory usage patterns reveal where `io.ReadAll` struggles:

- **For 1MB input**: `io.ReadAll` uses 5.2MB of memory, while the buffered approach uses only ~1MB
- **Memory efficiency**: The buffered approach consistently uses memory proportional to the input size
- **For typical API responses (1-10KB)**: Both approaches perform similarly with minimal overhead
- **The gap widens**: Performance differences become significant with data >100KB

### Allocations Tell the Story

- **io.ReadAll**: Allocation count grows with data size (32 allocations for 1MB)
- **Buffered approach**: Consistently only 3 allocations regardless of data size

## Why io.ReadAll Performs So Poorly

The performance difference isn't accidental—it's architectural. Here's what's happening under the hood:

### The io.ReadAll Growth Strategy

*(Based on Go 1.24.2 source code: `/usr/local/go/src/io/io.go`)*

When `io.ReadAll` doesn't know the final size of the data (which is almost always the case), it follows this pattern:

```go
func ReadAll(r Reader) ([]byte, error) {
	b := make([]byte, 0, 512)  // Start with 512-byte capacity
	for {
		n, err := r.Read(b[len(b):cap(b)])  // Read into available space
		b = b[:len(b)+n]                    // Extend slice to include new data
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err  // Done reading
		}

		if len(b) == cap(b) {               // Buffer is full, need more space
			b = append(b, 0)[:len(b)]     // Grow capacity, then restore length
		}
	}
}
```

The growth cycle works like this:
1. **Initial allocation**: 512 bytes capacity, 0 length
2. **Read data**: Into the available capacity (`b[len(b):cap(b)]`)
3. **Extend slice**: Update length to include new data (`b = b[:len(b)+n]`)
4. **Check if full**: When `len(b) == cap(b)`, no more space available
5. **Grow capacity**: `append(b, 0)` triggers Go's slice growth (typically doubling)
6. **Restore length**: `[:len(b)]` removes the extra byte, keeping only actual data
7. **Repeat**: Continue reading into the new capacity

This explains:
- **Exponential memory usage**: Each `append` reallocation creates temporary copies of all existing data
- **High allocation count**: Multiple buffer reallocations as data grows beyond each capacity threshold
- **Performance degradation**: Repeated copying operations with each growth phase

### The bytes.Buffer Growth Strategy

The buffered approach using `bytes.Buffer` with `io.Copy` is fundamentally different:

**bytes.Buffer Growth Algorithm:** *(Based on Go 1.24.2 source code: `/usr/local/go/src/bytes/buffer.go`)*
- Starts with a nil slice (no initial allocation)
- For initial small allocations (≤64 bytes), allocates exactly what's needed with 64-byte capacity (`smallBufferSize = 64`)
- Has optimizations to avoid unnecessary allocations:
  - Can grow by reslicing existing capacity when possible (`tryGrowByReslice`)
  - Can slide data down instead of reallocating when there's enough capacity
- When new allocation is needed, uses `growSlice` which typically doubles capacity (`c = 2 * cap(b)`)
- The growth pattern is more conservative and optimized than naive doubling

**The io.Copy Factor:** *(Based on Go standard library implementation)*
- `io.Copy` uses a 32KB buffer (`const copyBuffer = 32 * 1024`)
- Data is read and written in 32KB chunks
- This chunked approach means `bytes.Buffer` grows in predictable increments
- The buffer doesn't need to guess the final size—it grows incrementally as data arrives
- Each growth decision is made with concrete information about actual data size

This combination:
- Minimizes unnecessary reallocations
- Avoids the repeated copy penalty of unknown-size scenarios
- Maintains consistent allocation patterns
- Benefits from Go's optimized slice growth algorithms

### Further Optimization with io.CopyBuffer

**How io.CopyBuffer + sync.Pool Works:**
- `io.Copy` internally allocates a 32KB buffer for each operation
- `io.CopyBuffer` lets you provide a buffer (of your desired size), avoiding this allocation
- `sync.Pool` provides efficient buffer reuse across goroutines
- The combination eliminates temporary allocations entirely for the copy operation
- Only the final `bytes.Buffer` growth allocations remain

This optimization is particularly valuable in high-throughput scenarios where the same operation is performed repeatedly, such as web servers processing many requests.

## Real-World Implications

This isn't just academic—these performance characteristics have real implications:

### HTTP Response Handling

```go
// Avoid this for large responses
resp, _ := http.Get("https://api.example.com/large-dataset")
body, _ := io.ReadAll(resp.Body) // Potential performance killer

// Preferred approach
var buf bytes.Buffer
io.Copy(&buf, resp.Body)
body := buf.Bytes()
```

We do not know the size of the packet the server is sending to us and we do not want to run out of memory given that we know io.ReadAll is not memory efficient for large datasets.

### File Processing

When processing files of unknown size, the buffered approach can save significant memory and CPU cycles. `io.Copy` ensures we can have significantly less system calls as it reads
the data in 32kb chunks.

### Memory-Constrained Environments

In containerized environments or systems with limited memory, the memory overhead of `io.ReadAll` for large payloads could contribute to memory pressure, especially under high load with many concurrent operations processing large data streams.

### The 32KB Buffer Sweet Spot

The `io.Copy` function's 32KB buffer size isn't arbitrary—it's optimized for:
- **System call efficiency**: Reduces the number of read/write system calls
- **Memory locality**: 32KB fits comfortably in CPU caches
- **Network efficiency**: Aligns well with typical network packet sizes
- **Storage optimization**: Works efficiently with most storage block sizes

However, this also means:
- **Minimum memory overhead**: Even tiny reads will allocate at least 32KB temporarily
- **Chunked growth**: `bytes.Buffer` grows in response to actual data, not speculation
- **Predictable behavior**: Growth patterns are more deterministic than `io.ReadAll`'s guessing game

## When to Use Each Approach

### Use io.ReadAll when (most common cases):
- Processing typical API responses (<50KB)
- Working with small, bounded data sizes
- Prototyping or development where convenience matters
- Applications where the performance difference doesn't impact user experience
- **Rule of thumb**: If your data is consistently under 100KB, `io.ReadAll` is usually fine

### Use the buffered approach when:
- Processing file uploads or downloads
- Handling data streams of unknown or potentially large size (>100KB)
- Building systems that need to handle variable payload sizes efficiently
- Memory efficiency is critical for your use case
- Processing many large payloads concurrently

### Use the io.CopyBuffer optimization when:
- Building high-throughput systems processing large data repeatedly
- Every allocation matters for your performance profile (rare)
- You've profiled your application and identified this as a bottleneck
- Processing thousands of large payloads per second

## Key Takeaways

1. **Context matters**: `io.ReadAll` is fine for most use cases, but problematic for large or variable-sized data
2. **Size thresholds**: Performance differences become significant around 100KB+ data sizes
3. **Memory scaling**: The buffered approach scales memory usage linearly, while `io.ReadAll` can use 2-5x more memory
4. **Know your data patterns**: For typical web APIs and small files, the convenience of `io.ReadAll` usually outweighs the performance cost
5. **Optimize when needed**: Use the buffered approach when processing large or unknown-sized data streams
6. **Profile before optimizing**: Measure whether this optimization actually impacts your application's performance
7. **Further optimization exists**: `io.CopyBuffer` with `sync.Pool` provides additional benefits for high-throughput scenarios

## Conclusion

`io.ReadAll` remains a perfectly reasonable choice for most Go applications processing typical data sizes. However, when building systems that handle large files, variable-sized payloads, or operate under memory constraints, understanding these performance characteristics becomes important.

The key is recognizing when you've moved from the "small, predictable data" use case to the "large or unknown-sized data" scenario. For file uploads, data processing pipelines, or APIs that might return large responses, the buffered approach provides better memory efficiency and more predictable performance.

As with most performance optimizations, measure first, optimize second. If you're processing mostly small data, stick with `io.ReadAll` for its simplicity. If you're dealing with large or variable data sizes, consider the buffered approach as a better foundation for scalable systems.

---

*The complete benchmark code and examples from this article are available in the accompanying [repository](https://github.com/ekediala/copying_data).*
# copying_data

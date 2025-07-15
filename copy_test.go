package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

const (
	KB = 1024
	MB = 1024 * KB
)

func createTempFile(size int) (f *os.File, err error) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}

	file, err := os.CreateTemp(dir, "reader.ext")
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf("%w: closing file: %w", err, closeErr)
			}

			if removeErr := file.Close(); removeErr != nil {
				err = fmt.Errorf("%w: removing test directory: %w", err, removeErr)
			}
		}
	}()

	// Write data to the file.
	w := bufio.NewWriter(file)
	for range size {
		_, err = fmt.Fprintln(w, "just some bullshit")
		if err != nil {
			return nil, fmt.Errorf("writing data to temp file: %w", err)
		}
	}

	err = w.Flush()
	if err != nil {
		return nil, fmt.Errorf("writing data to temp file: %w", err)
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seeking to start of temp file: %w", err)
	}

	return file, nil
}

func runCopyTest(t *testing.T, copyFunc func(io.Reader, io.Writer) error, fileSize int) {
	t.Helper()

	reader, err := createTempFile(fileSize)
	if err != nil {
		t.Fatalf("Failed to create temp reader file: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(reader.Name()))
	defer reader.Close()

	writer, err := createTempFile(0)
	if err != nil {
		t.Fatalf("Failed to create temp writer file: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(writer.Name()))
	defer writer.Close()

	err = copyFunc(reader, writer)
	if err != nil {
		t.Fatalf("Copy function failed: %v", err)
	}

	readerInfo, err := reader.Stat()
	if err != nil {
		t.Fatalf("Failed to get reader file info: %v", err)
	}

	writerInfo, err := writer.Stat()
	if err != nil {
		t.Fatalf("Failed to get writer file info: %v", err)
	}

	if readerInfo.Size() != writerInfo.Size() {
		t.Fatalf("File sizes do not match. Expected %d, got %d", readerInfo.Size(), writerInfo.Size())
	}
}

func TestCopyWithBuffer(t *testing.T) {
	fileSizes := []int{KB, 10 * KB, MB}
	for _, size := range fileSizes {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			runCopyTest(t, CopyWithBuffer, size)
		})

	}
}

func TestCopyWithoutBuffer(t *testing.T) {
	fileSizes := []int{KB, 10 * KB, MB}
	for _, size := range fileSizes {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			runCopyTest(t, CopyWithoutBuffer, size)
		})
	}
}

func runBenchmark(b *testing.B, copyFunc func(io.Reader, io.Writer) error, fileSize int) {
	b.Helper()

	reader, err := createTempFile(fileSize)
	if err != nil {
		b.Fatalf("Failed to create temp reader file: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(reader.Name()))
	defer reader.Close()

	b.ResetTimer()
	for b.Loop() {
		reader.Seek(0, io.SeekStart) // Reset reader to the beginning of the file
		writer, err := createTempFile(0)
		if err != nil {
			b.Fatalf("Failed to create temp writer file: %v", err)
		}
		err = copyFunc(reader, writer)
		writer.Close()
		os.RemoveAll(filepath.Dir(writer.Name()))

		if err != nil {
			b.Fatalf("Copy function failed: %v", err)
		}
	}
}

func BenchmarkCopyWithBufferKB(b *testing.B) {
	runBenchmark(b, CopyWithBuffer, KB)
}

func BenchmarkCopyWithBuffer10KB(b *testing.B) {
	runBenchmark(b, CopyWithBuffer, 10*KB)
}

func BenchmarkCopyWithBufferMB(b *testing.B) {
	runBenchmark(b, CopyWithBuffer, MB)
}

func BenchmarkCopyWithoutBufferKB(b *testing.B) {
	runBenchmark(b, CopyWithoutBuffer, KB)
}

func BenchmarkCopyWithoutBuffer10KB(b *testing.B) {
	runBenchmark(b, CopyWithoutBuffer, 10*KB)
}

func BenchmarkCopyWithoutBufferMB(b *testing.B) {
	runBenchmark(b, CopyWithoutBuffer, MB)
}

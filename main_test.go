package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestReadWithIoReadAll(t *testing.T) {
	data := make([]byte, 10000)
	r := bytes.NewReader(data)
	if _, err := readBodyIoReadAll(r); err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
}

func TestReadWithBuffer(t *testing.T) {
	data := make([]byte, 10000)
	r := bytes.NewReader(data)
	if _, err := readBodyBuffered(r); err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
}

func BenchmarkReadMethods(b *testing.B) {
	sizes := []int{1024, 10000, 100000, 1000000}

	for _, size := range sizes {
		data := make([]byte, size)

		b.Run(fmt.Sprintf("IoReadAll_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				r := bytes.NewReader(data)
				_, err := readBodyIoReadAll(r)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Buffered_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				r := bytes.NewReader(data)
				_, err := readBodyBuffered(r)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

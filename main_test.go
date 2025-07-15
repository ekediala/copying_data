package main

import (
	"fmt"
	"os"
	"testing"
)

func TestReadWithIoReadAll(t *testing.T) {
	f, err := os.Open("SmartRaise_Dash.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := readBodyIoReadAll(f); err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
}

func TestReadWithBuffer(t *testing.T) {
	f, err := os.Open("SmartRaise_Dash.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readBodyBuffered(f); err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
}

func BenchmarkReadMethods(b *testing.B) {
	fileNames := []string{"SmartRaise_Dash.jpg", "README.md"}

	for _, fileName := range fileNames {
		data, err := os.Open(fileName)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(fmt.Sprintf("IoReadAll_%s", fileName), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := readBodyIoReadAll(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Buffered_%s", fileName), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, err := readBodyBuffered(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

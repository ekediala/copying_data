package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestReadWithIoReadAll(t *testing.T) {
	f, err := os.Open("SmartRaise_Dash.jpg")
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := readBodyIoReadAll(f, buf); err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
}

func TestReadWithBuffer(t *testing.T) {
	f, err := os.Open("SmartRaise_Dash.jpg")
	if err != nil {
		t.Fatal(err)
	}

	w, err := os.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	defer os.Remove("test.txt")

	if err := readBodyBuffered(f, w); err != nil {
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
			w, err := os.Create("test.txt")
			if err != nil {
				b.Fatal(err)
			}
			defer w.Close()
			defer os.Remove("test.txt")

			b.ReportAllocs()
			for b.Loop() {
				err := readBodyIoReadAll(data, w)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("io.Copy_%s", fileName), func(b *testing.B) {
			w, err := os.Create("test.txt")
			if err != nil {
				b.Fatal(err)
			}
			defer w.Close()
			defer os.Remove("test.txt")

			b.ReportAllocs()
			for b.Loop() {
				err := readBodyBuffered(data, w)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("io.CopyBuffer_%s", fileName), func(b *testing.B) {
			w, err := os.Create("test.txt")
			if err != nil {
				b.Fatal(err)
			}
			defer w.Close()
			defer os.Remove("test.txt")

			b.ReportAllocs()
			for b.Loop() {
				err := readBodyBuffered(data, w)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

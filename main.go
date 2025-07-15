package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"sync"
)

var bufBool = sync.Pool{
	New: func() any {
		return make([]byte, 32*1024)
	},
}

func readBodyIoReadAll(r io.Reader, w io.Writer) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading data: %w", err)
	}

	// process data. we just use a write here as a placeholder
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("writing data data: %w", err)
	}

	return nil
}

func readBodyBufferedPool(r io.Reader, w io.Writer) error {
	buf := bufBool.Get().([]byte)
	defer bufBool.Put(buf)

	_, err := io.CopyBuffer(w, r, buf)
	if err != nil {
		return fmt.Errorf("reading data: %w", err)
	}
	return nil
}

func readBodyBuffered(r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("reading data: %w", err)
	}
	return nil
}

func main() {
	var useBuffer bool
	var useSyncPool bool
	var memprofile string

	flag.BoolVar(&useBuffer, "b", false, "should we stream response body?")
	flag.BoolVar(&useSyncPool, "c", false, "should we use copy buffer?")
	flag.StringVar(&memprofile, "memprofile", "", "write memory profile to file")
	flag.Parse()

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		defer pprof.WriteHeapProfile(f)
	}

	f, err := os.Open("SmartRaise_Dash.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w, err := os.Create("test.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()
	defer os.Remove("test.txt")

	if useBuffer {
		err = readBodyBuffered(f, w)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if useSyncPool {
		err = readBodyBufferedPool(f, w)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	err = readBodyIoReadAll(f, w)
	if err != nil {
		log.Fatal(err)
	}

}

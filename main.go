package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func readBodyIoReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

func readBodyBuffered(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, fmt.Errorf("reading data: %w", err)
	}
	return buf.Bytes(), nil
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	var useBuffer bool
	flag.BoolVar(&useBuffer, "b", false, "should we stream response body?")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/", nil)
	if err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		Timeout: time.Minute,
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	if useBuffer {
		b, err := readBodyIoReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
		return
	}

	b, err := readBodyBuffered(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

}

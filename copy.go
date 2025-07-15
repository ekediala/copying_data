package main

import (
	"fmt"
	"io"
)

func CopyWithBuffer(r io.Reader, w io.Writer) (err error) {
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("writing: %w", err)
	}
	return nil
}

func CopyWithoutBuffer(r io.Reader, w io.Writer) error {
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		if _, err := w.Write(buf[:n]); err != nil {
			return fmt.Errorf("writing to writer: %w", err)
		}
	}
}

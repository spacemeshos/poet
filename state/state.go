package state

import (
	"bytes"
	"fmt"
	"os"

	"github.com/natefinch/atomic"
	xdr "github.com/nullstyle/go-xdr/xdr3"
)

func Persist(filename string, v any) error {
	var w bytes.Buffer
	_, err := xdr.Marshal(&w, v)
	if err != nil {
		return fmt.Errorf("serializing: %w", err)
	}

	err = atomic.WriteFile(filename, &w)
	if err != nil {
		return fmt.Errorf("writing to disk: %w", err)
	}

	return nil
}

func Load(filename string, v any) error {
	f, err := os.Open(filename) //#nosec G304
	if err != nil {
		return fmt.Errorf("loading file: %w", err)
	}
	defer f.Close()

	_, err = xdr.Unmarshal(f, v)
	if err != nil {
		return fmt.Errorf("deserializing: %w", err)
	}

	return nil
}

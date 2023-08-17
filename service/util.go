package service

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/natefinch/atomic"
	xdr "github.com/nullstyle/go-xdr/xdr3"
)

var ErrFileIsMissing = errors.New("file is missing")

func persist(filename string, v any) error {
	var w bytes.Buffer
	_, err := xdr.Marshal(&w, v)
	if err != nil {
		return fmt.Errorf("serialization failure: %v", err)
	}

	err = atomic.WriteFile(filename, &w)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil
}

func load(filename string, v any) error {
	data, err := os.ReadFile(filename) //#nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %v", ErrFileIsMissing, filename)
		}

		return fmt.Errorf("failed to read file: %v", err)
	}

	_, err = xdr.Unmarshal(bytes.NewReader(data), v)
	if err != nil {
		return fmt.Errorf("failed to deserialize: %v", err)
	}

	return nil
}

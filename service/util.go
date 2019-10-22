package service

import (
	"bytes"
	"fmt"
	"github.com/nullstyle/go-xdr/xdr3"
	"github.com/spacemeshos/poet/shared"
	"io/ioutil"
	"os"
)

func persist(filename string, v interface{}) error {
	var w bytes.Buffer
	_, err := xdr.Marshal(&w, v)
	if err != nil {
		return fmt.Errorf("serialization failure: %v", err)
	}

	err = ioutil.WriteFile(filename, w.Bytes(), shared.OwnerReadWrite)
	if err != nil {
		return fmt.Errorf("write to disk failure: %v", err)
	}

	return nil

}

func load(filename string, v interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file is missing: %v", filename)
		}

		return fmt.Errorf("failed to read file: %v", err)
	}

	_, err = xdr.Unmarshal(bytes.NewReader(data), v)
	if err != nil {
		return fmt.Errorf("failed to deserialize: %v", err)
	}

	return nil
}

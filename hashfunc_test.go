package main

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/spacemeshos/sha256-simd"
)

func BenchmarkSha256(t *testing.B) {
	buff := bytes.Buffer{}
	buff.Write([]byte("Seed data goes here"))
	out := [32]byte{}
	n := (uint64(1) << 20) * 101

	fmt.Printf("Computing %d serial sha-256s...\n", n)

	t1 := time.Now()

	for i := uint64(0); i < n; i++ {
		out = sha256.Sum256(buff.Bytes())
		buff.Reset()
		buff.Write(out[:])
	}

	e := time.Since(t1)
	r := uint64(float64(n) / e.Seconds())
	fmt.Printf("Final hash: %x. Running time: %s secs. Hash-rate: %d hashes-per-sec\n", buff.Bytes(), e, r)
}

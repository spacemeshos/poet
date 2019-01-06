package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/spacemeshos/poet-ref/shared"
	"github.com/spacemeshos/sha256-simd"
	"math"
	"testing"
	"time"
)

func BenchmarkSha256(t *testing.B) {
	buff := bytes.Buffer{}
	buff.Write([]byte("Seed data goes here"))
	out := [32]byte{}
	n := uint64(math.Pow(10, 8))

	fmt.Printf("Computing %d serial sha-256s...\n", n)

	t1 := time.Now().Unix()

	for i := uint64(0); i < n; i++ {
		out = sha256.Sum256(buff.Bytes())
		buff.Reset()
		buff.Write(out[:])
	}

	d := time.Now().Unix() - t1
	r := n / uint64(d)

	fmt.Printf("Final hash: %x. Running time: %d secs. Hash-rate:%d hashes-per-sec\n", buff.Bytes(), d, r)
}

func BenchmarkScrypt(t *testing.B) {
	x := make([]byte, 32)
	_, err := rand.Read(x)
	if err != nil {
		panic(err)
	}

	io := make([]byte, 32)
	_, err = rand.Read(io)
	if err != nil {
		panic(err)
	}

	n := 1000000
	hash := shared.NewScryptHashFunc(x)
	fmt.Printf("Computing %d serial scrypts...\n", n)
	t1 := time.Now().Unix()
	for i := 0; i < n; i++ {
		io = hash.HashSingle(io)
	}
	d := time.Now().Unix() - t1
	r := int64(n) / d
	fmt.Printf("Final hash: %x. Running time: %d secs. Hash-rate: %d hashes-per-sec\n", io, d, r)
}

package main

import (
	"github.com/minio/sha256-simd" // simd optimized sha256 computation
	//"crypto/sha256" // use the go crypto lib for comparison
	"bytes"
	"fmt"
	"math"
	"runtime"
	"time"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	buff := bytes.Buffer{}
	buff.Write([]byte("Seed data goes here"))
	out := [32]byte{}
	n := uint64(math.Pow(10, 9))

	fmt.Printf("Computing %d serial sha-256s...\n", n)

	t1 := time.Now().Unix()

	for i := uint64(0); i < n; i++ {
		out = sha256.Sum256(buff.Bytes())
		buff.Reset()
		buff.Write(out[:])
	}

	d := time.Now().Unix() - t1
	r := n / uint64(d)

	// my 2016 mbp w 2.5 GHz Intel Core i7 does ~3m hashes per second
	fmt.Printf("Final hash: %x took: %d secs. Hash-rate:%d hashes-per-sec\n", buff.Bytes(), d, r)
}

package main

import (
	"bytes"
	"fmt"
	"github.com/minio/sha256-simd"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"math"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// BenchmarkSha256()
	Playground()
}

func Playground() {

	const x = "this is a commitment"
	const phi = "this is a test root label"
	const n = 63

	v, err := internal.NewVerifier([]byte(x), n)
	if err != nil {
		println(err)
		return
	}

	c, err := v.CreteNipChallenge([]byte(phi))
	if err != nil {
		println(err)
		return
	}

	if len(c.Data) != shared.T  {
		println ("Expected t identifiers in challenge")
		return
	}

	println("Printing NIP challenge...")
	for idx, id := range c.Data {
		if len(id) != n {
			println("Unexpected identifier width")
			return
		}

		println(idx, id)
	}
}

func BenchmarkSha256() {
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

	// my 2016 mbp w 2.5 GHz Intel Core i7 does ~3m hashes per second
	fmt.Printf("Final hash: %x took: %d secs. Hash-rate:%d hashes-per-sec\n", buff.Bytes(), d, r)
}
package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/minio/sha256-simd"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"math"
	"os"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//BenchmarkSha256()
	Playground()
}

func Playground() {

	x := make([]byte, 32)
	_, err := rand.Read(x)
	if err != nil {
		os.Exit(-1)
	}

	const n = 15

	p, err := internal.NewProver(x, n)

	defer p.DeleteStore()

	if err != nil {
		println("Failed to create prover.")
		os.Exit(-1)
	}

	println("Computing dag...")

	t1 := time.Now().Unix()

	p.ComputeDag(func(phi shared.Label, err error) {
		fmt.Printf("Dag root label: %s\n", internal.GetDisplayValue(phi))
		if err != nil {
			println("Failed to compute dag.")
			os.Exit(-1)
		}

		proof, err := p.GetNonInteractiveProof()
		if err != nil {
			println("Failed to create NIP.")
			os.Exit(-1)
		}

		v, err := internal.NewVerifier(x, n)
		if err != nil {
			println("Failed to create verifier.")
			os.Exit(-1)
		}

		c, err := v.CreteNipChallenge(proof.Phi)
		if err != nil {
			println("Failed to create NIP challenge.")
			os.Exit(-1)
		}

		res := v.Verify(c, proof)
		if res == false {
			println("Failed to verify NIP proof.")
			os.Exit(-1)
		}

		c1, err := v.CreteRndChallenge()
		if err != nil {
			println("Failed to create rnd challenge.")
			os.Exit(-1)
		}

		proof1, err := p.GetProof(c1)

		res = v.Verify(c1, proof1)
		if res == false {
			println("Failed to verify interactive proof.")
			os.Exit(-1)
		}

		d := time.Now().Unix() - t1

		fmt.Printf("Proof generated in %d seconds.\n", d)

	})

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

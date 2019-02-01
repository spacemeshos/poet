package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"runtime"
	"time"
)

var (
	rpcPort    uint
	wProxyPort uint
	n          uint
)

func init() {

	// disable memory profiling which cause memory to grow unbounded
	runtime.MemProfileRate = 0

	flag.UintVar(&n, "n", 10, "Table size = 2^n")

	flag.UintVar(&rpcPort, "rpcport", 50052, "RPC server listening port")
	flag.UintVar(&wProxyPort, "wproxyport", 8080, "RPC server web proxy listening port")
	flag.Parse()
}

func main() {
	x := make([]byte, 32)

	_, err := rand.Read(x)
	if err != nil {
		panic("no entropy")
	}

	p, err := internal.NewProver(x, n, shared.NewHashFunc(x))
	// defer p.DeleteStore()
	if err != nil {
		panic("can't create prover")
	}

	println("Computing dag...")
	t1 := time.Now()

	phi, err := p.ComputeDag()
	if err != nil {
		panic("Can't compute dag")
	}

	e := time.Since(t1)
	fmt.Printf("Proof generated in %s (%f)\n", e, e.Seconds())
	fmt.Printf("Dag root label: %s\n", internal.GetDisplayValue(phi))

	proof, err := p.GetNonInteractiveProof()
	if err != nil {
		panic("Failed to create nip")
	}

	v, err := internal.NewVerifier(x, n, shared.NewHashFunc(x))
	if err != nil {
		panic("Failed to create verifier")
	}

	t1 = time.Now()
	verified, err := v.VerifyNIP(proof)
	if err != nil {
		panic("Failed to verify nip")
	}
	e = time.Since(t1)
	if !verified {
		panic("Failed to verify proof")
	}

	fmt.Printf("Proof verified in %s (%f)\n", e, e.Seconds())

}

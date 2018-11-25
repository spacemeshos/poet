package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/minio/sha256-simd"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/service"
	"github.com/spacemeshos/poet-ref/shared"
	"log"
	"math"
	"os"
	"time"
)

var (
	n   uint
	rpc bool
)

func init() {
	flag.BoolVar(&rpc, "rpc", false, "start RPC server")
	flag.UintVar(&n, "n", 1, "time parameter. Shared between verifier and prover")
	flag.Parse()
	fmt.Printf("n = %d\n", n)
}

func main() {
	if rpc {
		if err := service.Start(); err != nil {
			log.Fatal(err)
		}
	} else {
		//	runtime.GOMAXPROCS(runtime.NumCPU())
		//BenchmarkSha256()
		//BenchmarkScrypt()
		Playground()
	}
}

func Playground() {
	x := make([]byte, 32)
	_, err := rand.Read(x)
	if err != nil {
		panic(err)
	}

	p, err := internal.NewProver(x, n, shared.NewScryptHashFunc(x))

	defer p.DeleteStore()

	if err != nil {
		println("Failed to create prover.")
		os.Exit(-1)
	}

	println("Computing dag...")

	t1 := time.Now().Unix()

	phi, err := p.ComputeDag()

	d := time.Now().Unix() - t1

	fmt.Printf("Proof generated in %d seconds.\n", d)

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

	v, err := internal.NewVerifier(x, n, shared.NewScryptHashFunc(x))
	if err != nil {
		println("Failed to create verifier.")
		os.Exit(-1)
	}

	a, err := v.VerifyNIP(proof)
	println(a)

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

	d1 := time.Now().Unix() - t1

	fmt.Printf("Proof generated and verified in %d seconds.\n", d1)

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

	fmt.Printf("Final hash: %x. Running time: %d secs. Hash-rate:%d hashes-per-sec\n", buff.Bytes(), d, r)
}

func BenchmarkScrypt() {

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

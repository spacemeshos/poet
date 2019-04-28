package main

import (
	"crypto/rand"
	"fmt"
	"github.com/spacemeshos/poet/internal"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
	"runtime"

	"log"
	"os"
	"path"
	"runtime/pprof"
	"time"
)

func main() {
	runtime.MemProfileRate = 0
	println("Memory profiling disabled.")

	cfg, err := loadConfig()
	if err != nil {
		os.Exit(1)
	}

	// Enable memory profiling
	if cfg.CPU {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal("cant get current dir", err)
		}

		profFilePath := path.Join(dir, "./CPU.prof")
		fmt.Printf("CPU profile: %s\n", profFilePath)

		f, err := os.Create(profFilePath)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()

		println("Cpu profiling enabled and started...")
	}

	x := make([]byte, 20)
	_, err = rand.Read(x)
	if err != nil {
		panic("no entropy")
	}

	p, err := prover.New(x, cfg.N, shared.NewHashFunc(x))
	defer p.DeleteStore()
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

	v, err := verifier.New(x, cfg.N, shared.NewHashFunc(x))
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

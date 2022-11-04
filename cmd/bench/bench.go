package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
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

	challenge := make([]byte, 20)
	_, err = rand.Read(challenge)
	if err != nil {
		panic("no entropy")
	}

	end := time.Now().Add(cfg.Duration)
	securityParam := shared.T

	t1 := time.Now()
	println("Computing dag...")
	tempdir, _ := os.MkdirTemp("", "poet-test")
	leafs, merkleProof, err := prover.GenerateProofWithoutPersistency(context.Background(), tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), end, securityParam, prover.LowestMerkleMinMemoryLayer)
	if err != nil {
		panic("failed to generate proof")
	}

	e := time.Since(t1)
	fmt.Printf("Proof from %d leafs generated in %s (%f)\n", leafs, e, e.Seconds())
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)

	t1 = time.Now()
	err = verifier.Validate(*merkleProof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), leafs, securityParam)
	if err != nil {
		panic("Failed to verify nip")
	}

	e = time.Since(t1)
	fmt.Printf("Proof verified in %s (%f)\n", e, e.Seconds())
}

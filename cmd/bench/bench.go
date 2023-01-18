package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

const (
	cpuProfFilePath = "./cpu.prof"
	memProfFilePath = "./mem.prof"
)

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cfg, err := loadConfig()
	if err != nil {
		os.Exit(1)
	}

	// Enable cpu profiling
	if cfg.CPU {
		fmt.Printf("CPU profile: %s\n", cpuProfFilePath)
		f, err := os.Create(cpuProfFilePath)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
		defer func() { _ = f.Close() }()
		defer println("closed proifiling")
		println("Cpu profiling enabled and started...")
	}

	challenge := []byte("1234567890abcdefghij")
	securityParam := shared.T

	tempdir, _ := os.MkdirTemp("", "poet-test")
	proofGenStarted := time.Now()
	end := proofGenStarted.Add(cfg.Duration)
	leafs, merkleProof, err := prover.GenerateProofWithoutPersistency(
		context.Background(),
		tempdir,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		end,
		securityParam,
		prover.LowestMerkleMinMemoryLayer,
	)
	pprof.StopCPUProfile()
	if err != nil {
		panic(fmt.Sprintf("failed to generate proof: %v", err))
	}

	fmt.Printf("Memory profile: %s\n", memProfFilePath)
	f, err := os.Create(memProfFilePath)
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer func() { _ = f.Close() }()
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}

	proofGenDuration := time.Since(proofGenStarted)
	fmt.Printf(
		"Proof with %d leafs generated in %s (%d/s)\n",
		leafs,
		proofGenDuration,
		leafs/uint64(proofGenDuration.Seconds()),
	)
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)

	proofValidStarted := time.Now()
	err = verifier.Validate(
		*merkleProof,
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		leafs,
		securityParam,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to verify nip: %v", err))
	}

	proofValidDuration := time.Since(proofValidStarted)
	fmt.Printf("Proof verified in %s\n", proofValidDuration)
}

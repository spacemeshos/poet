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

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cfg, err := loadConfig()
	if err != nil {
		os.Exit(1)
	}

	// Enable cpu profiling
	if cfg.CpuProfile != nil {
		fmt.Printf("Starting CPU profile: %s\n", *cfg.CpuProfile)
		f, err := os.Create(*cfg.CpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
		defer func() { _ = f.Close() }()
	}

	challenge := []byte("1234567890abcdefghij")
	securityParam := shared.T

	tempdir, _ := os.MkdirTemp("", "poet-test")
	defer os.RemoveAll(tempdir)

	proofGenStarted := time.Now()
	end := proofGenStarted.Add(cfg.Duration)
	leafs, merkleProof, err := prover.GenerateProofWithoutPersistency(
		context.Background(),
		prover.TreeConfig{
			Datadir: tempdir,
		},
		hash.GenLabelHashFunc(challenge),
		hash.GenMerkleHashFunc(challenge),
		end,
		securityParam,
	)
	pprof.StopCPUProfile()
	if err != nil {
		panic(fmt.Sprintf("failed to generate proof: %v", err))
	}

	if cfg.Memprofile != nil {
		fmt.Printf("Collecting memory profile to %s\n", *cfg.Memprofile)
		f, err := os.Create(*cfg.Memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer func() { _ = f.Close() }()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
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

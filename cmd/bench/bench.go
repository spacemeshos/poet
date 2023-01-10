package main

import (
	"context"
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
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	runtime.MemProfileRate = 0

	cfg, err := loadConfig()
	if err != nil {
		os.Exit(1)
	}

	// Enable cpu profiling
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

	challenge := []byte("1234567890abcdefghij")
	securityParam := shared.T

	tempdir, _ := os.MkdirTemp("", "poet-test")
	proofGenStarted := time.Now()
	end := proofGenStarted.Add(cfg.Duration)
	leafs, merkleProof, err := prover.GenerateProofWithoutPersistency(context.Background(), tempdir, challenge, hash.NewGenMerkleHashFunc(challenge), end, securityParam, 2)
	pprof.StopCPUProfile()
	if err != nil {
		panic(fmt.Sprintf("failed to generate proof: %v", err))
	}

	proofGenDuration := time.Since(proofGenStarted)
	fmt.Printf("Proof with %d leafs generated in %s (%d/s)\n", leafs, proofGenDuration, leafs/uint64(proofGenDuration.Seconds()))
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)

	proofValidStarted := time.Now()
	err = verifier.Validate(*merkleProof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), leafs, securityParam)
	if err != nil {
		panic(fmt.Sprintf("Failed to verify nip: %v", err))
	}

	proofValidDuration := time.Since(proofValidStarted)
	fmt.Printf("Proof verified in %s\n", proofValidDuration)
}

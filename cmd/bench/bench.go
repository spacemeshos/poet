package main

import (
	"crypto/rand"
	"fmt"
	"github.com/spacemeshos/poet/hash"
	"github.com/spacemeshos/poet/prover"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
	"io/ioutil"
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

	challenge := make([]byte, 20)
	_, err = rand.Read(challenge)
	if err != nil {
		panic("no entropy")
	}

	numLeaves := uint64(1) << cfg.N
	securityParam := shared.T
	fmt.Printf("numLeaves: %d, securityParam: %d\n", numLeaves, securityParam)

	t1 := time.Now()
	println("Computing dag...")
	tempdir, _ := ioutil.TempDir("", "poet-test")
	merkleProof, err := prover.GenerateProofWithoutPersistency(tempdir, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam, prover.LowestMerkleMinMemoryLayer)
	if err != nil {
		panic("failed to generate proof")
	}

	e := time.Since(t1)
	fmt.Printf("Proof generated in %s (%f)\n", e, e.Seconds())
	fmt.Printf("Dag root label: %x\n", merkleProof.Root)
	size := proofSize(merkleProof)
	fmt.Printf("Proof size: %s (%d bytes)\n", ByteCountIEC(size), size)

	t1 = time.Now()
	err = verifier.Validate(*merkleProof, hash.GenLabelHashFunc(challenge), hash.GenMerkleHashFunc(challenge), numLeaves, securityParam)
	if err != nil {
		panic("Failed to verify nip")
	}

	e1 := time.Since(t1)
	fmt.Printf("Proof verified in %s (%f)\n", e1, e1.Seconds())

	fmt.Printf("%d %d %f %f %d\n", numLeaves, securityParam, e.Seconds(), e1.Seconds(), size)
}

func proofSize(proof *shared.MerkleProof) int {
	size := 0
	for _, node := range proof.ProofNodes {
		size += len(node)
	}
	size += len(proof.Root)
	for _, leaf := range proof.ProvenLeaves {
		size += len(leaf)
	}
	return size
}

func ByteCountIEC(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

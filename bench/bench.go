package bench

import (
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"time"
)

var (
	n   uint // protocol n param
	cpu bool // cpu profiling
)

func init() {

	runtime.MemProfileRate = 0
	println("Memory profiling disabled.")

	flag.BoolVar(&cpu, "cpu", false, "profile cpu use")
	flag.UintVar(&n, "n", 10, "Table size = 2^n")
	flag.Parse()

}

func Bench() {

	if cpu { // enable memory profiling

		dir, err := os.Getwd()
		if err != nil {
			log.Fatal("cant get current dir", err)
		}

		profFilePath := path.Join(dir, "./cpu.prof")

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

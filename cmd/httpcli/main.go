package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spacemeshos/merkle-tree"
	"github.com/urfave/cli"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/spacemeshos/poet/cmd/httpcli/client"
	"github.com/spacemeshos/poet/hash"
	rpcapi "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/shared"
	"github.com/spacemeshos/poet/verifier"
)

func verify_proof_from_file(filename string) error {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	proof := &rpcapi.ProofResponse{}
	if err := protojson.Unmarshal(bytes, proof); err != nil {
		return fmt.Errorf("unmarshaling proof: %w", err)
	}
	proof.Proof.Members = [][]byte{}

	return verify_proof(proof.Proof)
}

func verify_proof(proof *rpcapi.PoetProof) error {
	merkleProof := shared.MerkleProof{
		Root:         proof.Proof.Root,
		ProvenLeaves: proof.Proof.ProvenLeaves,
		ProofNodes:   proof.Proof.ProofNodes,
	}

	tree, err := merkle.NewTreeBuilder().
		WithHashFunc(shared.HashMembershipTreeNode).
		Build()
	if err != nil {
		return fmt.Errorf("creating Merkle Tree: %w", err)
	}
	for _, member := range proof.Members {
		if err := tree.AddLeaf(member[:]); err != nil {
			return fmt.Errorf("adding leaf to Merkle Tree: %w", err)
		}
	}
	root := tree.Root()
	encoded := base64.StdEncoding.EncodeToString(root)
	fmt.Printf("calculated proof statement: %X (%s)\n", root, encoded)

	labelHashFunc := hash.GenLabelHashFunc(root)
	merkleHashFunc := hash.GenMerkleHashFunc(root)
	return verifier.Validate(merkleProof, labelHashFunc, merkleHashFunc, proof.Leaves, shared.T)
}

func proof(address, round string) error {
	fmt.Printf("Quering %s for proof for round %v\n", address, round)

	cl, err := client.NewHTTPPoetClient(address)
	// cl, err := activation.NewHTTPPoetClient(address, activation.DefaultPoetConfig())
	if err != nil {
		return err
	}
	proof, err := cl.Proof(context.Background(), round)
	// _, m, err := cl.Proof(context.Background(), round)
	if err != nil {
		if status.Code(err) == codes.Unavailable {
			fmt.Println("Unavailable")
			time.Sleep(time.Second)
		}
		return fmt.Errorf("getting proof: %v", err)
	}
	fmt.Printf("got proof with %.2f M leaves and %d members\n", float64(proof.Leaves)/1_000_000, len(proof.Members))

	if err := verify_proof(proof); err != nil {
		fmt.Printf("❌ failed: %v\n", err)
	} else {
		fmt.Println("✅ proof is valid")
	}
	fmt.Println("===================================")
	return nil
}

func spam_proofs(address string, start, count int) error {
	fmt.Printf("Quering for proofs for rounds %v-%v\n", start, start+count)

	cl, err := client.NewHTTPPoetClient(address)
	if err != nil {
		return err
	}

	for r := start; r < start+count; r++ {
		_, err := cl.Proof(context.Background(), strconv.Itoa(r))
		if err != nil {
			fmt.Printf("getting proof: %v", err)
		}

		fmt.Println("Got proof")
	}

	return nil
}

func submit(address string) error {
	cl, err := client.NewHTTPPoetClient(address)
	if err != nil {
		return err
	}

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %v", err)
	}

	challenge := []byte(fmt.Sprintf("challenge %v", pubKey))

	certPrivKeyB64 := os.Getenv("CERT_PRIVATE_KEY")
	certPrivKey, err := base64.StdEncoding.DecodeString(certPrivKeyB64)
	if err != nil {
		return fmt.Errorf("decoding cert private key: %v", err)
	}

	err = cl.Submit(
		context.Background(),
		[]byte{},
		challenge,
		ed25519.Sign(privKey, challenge),
		pubKey,
		client.PoetPoW{},
		ed25519.Sign(certPrivKey, pubKey),
	)
	if err != nil {
		return fmt.Errorf("submitting challenge: %w", err)
	}
	return nil
}

func info(address string) error {
	cl, err := client.NewHTTPPoetClient(address)
	if err != nil {
		return err
	}

	info, err := cl.PoetServiceID(context.Background())
	if err != nil {
		return fmt.Errorf("querying proof: %w", err)
	}
	fmt.Printf("PoetServiceID: %v\n", info)

	return nil
}

func main() {
	app := &cli.App{
		Commands: []cli.Command{
			{
				Name: "genkey",
				Action: func(cCtx *cli.Context) error {
					pubkey, priv, err := ed25519.GenerateKey(nil)
					if err != nil {
						return err
					}
					fmt.Printf("private key: %s\n", base64.StdEncoding.EncodeToString(priv))
					fmt.Printf("pub key: %s\n", base64.StdEncoding.EncodeToString(pubkey))
					return nil
				},
			},
			{
				Name: "info",
				Action: func(cCtx *cli.Context) error {
					return info(cCtx.Args().First())
				},
			},
			{
				Name: "submit",
				Action: func(cCtx *cli.Context) error {
					return submit(cCtx.Args().First())
				},
			},
			{
				Name: "proof",
				Action: func(cCtx *cli.Context) error {
					return proof(cCtx.Args().First(), cCtx.Args().Get(1))
				},
			},
			{
				Name: "get_proofs",
				Action: func(cCtx *cli.Context) error {
					for _, url := range []string{
						"http://poet-110.spacemesh.network",
						"http://poet-111.spacemesh.network",
						"http://poet-112.spacemesh.network",
						"http://mainnet-poet-0.spacemesh.network",
						"http://mainnet-poet-1.spacemesh.network",
					} {
						err := proof(url, cCtx.Args().Get(0))
						if err != nil {
							fmt.Printf("couldn't get poet proof from %s: %v", url, err)
						}
					}
					return nil
				},
			},
			{
				Name: "verify_proof",
				Action: func(cCtx *cli.Context) error {
					if err := verify_proof_from_file(cCtx.Args().First()); err != nil {
						fmt.Println("===================================")
						fmt.Printf(" failed: %v\n", err)
						fmt.Println("===================================")
						return err
					}
					fmt.Println("proof is valid")
					return nil
				},
			},
			{
				Name: "spam_proofs",
				Action: func(cCtx *cli.Context) error {
					start, err := strconv.Atoi(cCtx.Args().Get(1))
					if err != nil {
						return err
					}
					count, err := strconv.Atoi(cCtx.Args().Get(2))
					if err != nil {
						return err
					}
					return spam_proofs(cCtx.Args().First(), start, count)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

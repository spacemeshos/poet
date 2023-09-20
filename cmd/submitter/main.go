package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"time"

	grpcpool "github.com/processout/grpc-go-pool"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	apiv1 "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/shared"
)

func submit(pool *grpcpool.Pool) error {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %v", err)
	}

	ch := make([]byte, 32)
	_, _ = rand.Read(ch)

	clientconn, err := pool.Get(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get client: %v", err)
	}
	defer clientconn.Close()

	client := apiv1.NewPoetServiceClient(clientconn.ClientConn)

	resp, err := client.PowParams(context.Background(), &apiv1.PowParamsRequest{})
	if err != nil {
		return fmt.Errorf("failed to get pow params: %v", err)
	}

	nonce, err := shared.FindSubmitPowNonce(
		context.Background(),
		resp.PowParams.Challenge,
		ch,
		pubKey,
		uint(resp.PowParams.Difficulty),
	)
	if err != nil {
		return fmt.Errorf("failed to find nonce: %v", err)
	}

	signature := ed25519.Sign(privKey, ch)
	_, err = client.Submit(context.Background(), &apiv1.SubmitRequest{
		Nonce:     nonce,
		Challenge: ch,
		Pubkey:    pubKey,
		Signature: signature,
		PowParams: resp.PowParams,
	})
	return err
}

func main() {
	pool, err := grpcpool.New(func() (*grpc.ClientConn, error) {
		return grpc.Dial("localhost:50002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	}, 10000, 10000, time.Minute)
	if err != nil {
		panic(err)
	}

	var eg errgroup.Group
	for i := 0; i < 10000; i++ {
		eg.Go(func() error {
			for j := 0; j < 20; j++ {
				if err := submit(pool); err != nil {
					return err
				}
			}
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		panic(err)
	}
}

package broadcaster

import (
	"context"
	"fmt"
	"github.com/spacemeshos/poet/broadcaster/pb"
	"github.com/spacemeshos/smutil/log"
	"google.golang.org/grpc"
	"time"
)

const timeout = 30 * time.Second

type Broadcaster struct {
	grpcClient pb.SpacemeshServiceClient
}

func (b *Broadcaster) BroadcastProof(msg []byte) error {
	if b.grpcClient == nil {
		return nil
	}
	pbMsg := &pb.BinaryMessage{Data: msg}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	response, err := b.grpcClient.BroadcastPoet(ctx, pbMsg)
	if err != nil {
		return fmt.Errorf("failed to broadcast poet after %v: %v", time.Since(start), err)
	}
	if response.Value != "ok" {
		return fmt.Errorf("node responded after %v: %v", time.Since(start), response.Value)
	}
	log.Info("completed broadcast successfully after %v, proof size: %d", time.Since(start), len(msg))
	return nil
}

func New(target string) (*Broadcaster, error) {
	if target == "NO_BROADCAST" {
		log.Info("Broadcast is disabled")
		return &Broadcaster{}, nil
	}

	conn, err := newClientConn(target)
	if err != nil {
		return nil, err
	}

	return &Broadcaster{
		grpcClient: pb.NewSpacemeshServiceClient(conn),
	}, nil
}

// newClientConn returns a new gRPC client
// connection to the specified target.
func newClientConn(target string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rpc server: %v", err)
	}

	return conn, nil
}

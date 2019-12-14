package broadcaster

import (
	"context"
	"errors"
	"fmt"
	"github.com/spacemeshos/poet/broadcaster/pb"
	"github.com/spacemeshos/smutil/log"
	"google.golang.org/grpc"
	"sync"
	"time"
)

const (
	connectionTimeout = 10 * time.Second
	broadcastTimeout  = 30 * time.Second
)

type Broadcaster struct {
	clients     []pb.SpacemeshServiceClient
	connections []*grpc.ClientConn
}

func (b *Broadcaster) BroadcastProof(msg []byte, roundId string, members [][]byte) error {
	if b.clients == nil {
		log.Info("Broadcast is disabled, not broadcasting round %v proof", roundId)
		return nil
	}
	pbMsg := &pb.BinaryMessage{Data: msg}
	ctx, cancel := context.WithTimeout(context.Background(), broadcastTimeout)
	defer cancel()

	responses := make([]*pb.SimpleMessage, len(b.clients))
	errs := make([]error, len(b.clients))

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(len(b.clients))
	for i, client := range b.clients {
		i := i
		client := client
		go func() {
			defer wg.Done()
			responses[i], errs[i] = client.BroadcastPoet(ctx, pbMsg)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	// Parse and concatenate potential errors.
	var numErrors int
	var retErr error
	for i := range b.clients {
		err := errs[i]
		res := responses[i]
		if err != nil {
			err = fmt.Errorf("failed to broadcast via gateway node at \"%v\" after %v: %v", b.connections[i].Target(), elapsed, err)
		} else if res.Value != "ok" {
			err = fmt.Errorf("failed to broadcast via gateway node at \"%v\" after %v: node response: %v", b.connections[i].Target(), elapsed, res.Value)
		} else {
			// Valid response.
			continue
		}

		numErrors++
		if retErr == nil {
			retErr = err
		} else {
			retErr = fmt.Errorf("%v | %v", retErr, err)
		}
	}

	// Iff all requests failed, return an error.
	if numErrors == len(b.clients) {
		return retErr
	}

	// If some requests failed, log it.
	if retErr != nil {
		log.Warning("Round %v proof broadcast failed on %d gateway nodes: %v", numErrors, roundId, retErr)
	}

	log.Info("Round %v proof broadcast completed successfully after %v, num of members: %d, proof size: %d", roundId, elapsed, len(members), len(msg))
	return nil

}

func New(gatewayAddresses []string, disableBroadcast bool) (*Broadcaster, error) {
	if disableBroadcast {
		log.Info("Broadcast is disabled")
		return &Broadcaster{}, nil
	}

	if len(gatewayAddresses) == 0 {
		return nil, errors.New("gateway addresses were not provided")
	}

	connections := make([]*grpc.ClientConn, len(gatewayAddresses))
	errs := make([]error, len(gatewayAddresses))

	log.Info("Attempting to connect to Spacemesh gateway nodes at %v", gatewayAddresses)
	var wg sync.WaitGroup
	wg.Add(len(gatewayAddresses))
	for i, address := range gatewayAddresses {
		i := i
		address := address
		go func() {
			defer wg.Done()
			connections[i], errs[i] = newClientConn(address, connectionTimeout)
			if errs[i] == nil {
				log.Info("Successfully connected to Spacemesh gateway node at \"%v\"", address)
			}
		}()
	}
	wg.Wait()

	// If >0 errors were returned, return them concatenated and close the other successful connections.
	var retErr error
	for i, err := range errs {
		if err != nil {
			err := fmt.Errorf("failed to connect to Spacemesh gateway node at \"%v\": %v", gatewayAddresses[i], err)
			if retErr == nil {
				retErr = err
			} else {
				retErr = fmt.Errorf("%v | %v", retErr, err)
			}
		}
	}
	if retErr != nil {
		for _, conn := range connections {
			if conn != nil {
				_ = conn.Close()
			}
		}
		return nil, retErr
	}

	clients := make([]pb.SpacemeshServiceClient, len(connections))
	for i, conn := range connections {
		clients[i] = pb.NewSpacemeshServiceClient(conn)
	}

	return &Broadcaster{clients: clients, connections: connections}, nil
}

// newClientConn returns a new gRPC client
// connection to the specified target.
func newClientConn(target string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

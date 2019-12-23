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
	DefaultConnTimeout      = 10 * time.Second
	DefaultBroadcastTimeout = 30 * time.Second
)

type Broadcaster struct {
	clients               []pb.SpacemeshServiceClient
	connections           []*grpc.ClientConn
	broadcastTimeout      time.Duration
	broadcastAckThreshold uint
}

func New(gatewayAddresses []string, disableBroadcast bool, connTimeout time.Duration, connAcksThreshold uint, broadcastTimeout time.Duration, broadcastAcksThreshold uint) (*Broadcaster, error) {
	if disableBroadcast {
		log.Info("Broadcast is disabled")
		return &Broadcaster{}, nil
	}

	if len(gatewayAddresses) == 0 {
		return nil, errors.New("number of gateway addresses must be greater than 0")
	}
	if connAcksThreshold < 1 {
		return nil, errors.New("successful connections threshold must be greater than 0")
	}
	if broadcastAcksThreshold < 1 {
		return nil, errors.New("successful broadcast threshold must be greater than 0")
	}
	if len(gatewayAddresses) < int(connAcksThreshold) {
		return nil, fmt.Errorf("number of gateway addresses (%d) must be greater than the successful connections threshold (%d)", len(gatewayAddresses), connAcksThreshold)
	}
	if connAcksThreshold < broadcastAcksThreshold {
		return nil, fmt.Errorf("the successful connections threshold (%d) must be greater than the successful broadcast threshold (%d)", connAcksThreshold, broadcastAcksThreshold)
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
			connections[i], errs[i] = newClientConn(address, connTimeout)
			if errs[i] == nil {
				log.Info("Successfully connected to Spacemesh gateway node at \"%v\"", address)
			}
		}()
	}
	wg.Wait()

	// Count successful connections and concatenate errors of non-successful ones.
	var numAcks int
	var retErr error
	for i, err := range errs {
		if err == nil {
			numAcks++
			continue
		}

		err := fmt.Errorf("failed to connect to Spacemesh gateway node at \"%v\": %v", gatewayAddresses[i], err)
		if retErr == nil {
			retErr = err
		} else {
			retErr = fmt.Errorf("%v | %v", retErr, err)
		}
	}

	// If successful connections threshold wasn't met, return errors concatenation and close the other successful connections.
	if numAcks < int(connAcksThreshold) {
		for _, conn := range connections {
			if conn != nil {
				_ = conn.Close()
			}
		}
		return nil, retErr
	}

	// If some connections failed, log it.
	if retErr != nil {
		numErrors := len(gatewayAddresses) - numAcks
		log.Warning("Failed to connect to %d/%d gateway nodes: %v", numErrors, len(gatewayAddresses), retErr)
	}

	clients := make([]pb.SpacemeshServiceClient, len(connections))
	for i, conn := range connections {
		clients[i] = pb.NewSpacemeshServiceClient(conn)
	}

	return &Broadcaster{
		clients:               clients,
		connections:           connections,
		broadcastTimeout:      broadcastTimeout,
		broadcastAckThreshold: broadcastAcksThreshold,
	}, nil
}

func (b *Broadcaster) BroadcastProof(msg []byte, roundId string, members [][]byte) error {
	if b.clients == nil {
		log.Info("Broadcast is disabled, not broadcasting round %v proof", roundId)
		return nil
	}
	pbMsg := &pb.BinaryMessage{Data: msg}
	ctx, cancel := context.WithTimeout(context.Background(), b.broadcastTimeout)
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

	// Count successful responses and concatenate errors of non-successful ones.
	var numAcks int
	var retErr error
	for i := range b.clients {
		res := responses[i]
		err := errs[i]

		if err != nil {
			err = fmt.Errorf("failed to broadcast via gateway node at \"%v\" after %v: %v", b.connections[i].Target(), elapsed, err)
		} else if res.Value != "ok" {
			err = fmt.Errorf("failed to broadcast via gateway node at \"%v\" after %v: node response: %v", b.connections[i].Target(), elapsed, res.Value)
		} else {
			// Valid response.
			numAcks++
			continue
		}

		if retErr == nil {
			retErr = err
		} else {
			retErr = fmt.Errorf("%v | %v", retErr, err)
		}
	}

	// If successful broadcasts threshold wasn't met, return errors concatenation.
	if numAcks < int(b.broadcastAckThreshold) {
		return retErr
	}

	// If some requests failed, log it.
	if retErr != nil {
		numErrors := len(b.clients) - numAcks
		log.Warning("Round %v proof broadcast failed on %d/%d gateway nodes: %v", numErrors, len(b.clients), roundId, retErr)
	}

	log.Info("Round %v proof broadcast completed successfully after %v, num of members: %d, proof size: %d", roundId, elapsed, len(members), len(msg))
	return nil
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

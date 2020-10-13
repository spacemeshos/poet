package broadcaster

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/spacemeshos/api/release/go/spacemesh/v1"
	"github.com/spacemeshos/smutil/log"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"sync"
	"time"
)

const (
	DefaultConnTimeout      = 10 * time.Second
	DefaultBroadcastTimeout = 30 * time.Second
)

// Broadcaster is responsible for broadcasting proofs to a list of Spacemesh-compatible gateway nodes.
type Broadcaster struct {
	clients               []pb.GatewayServiceClient
	connections           []*grpc.ClientConn
	broadcastTimeout      time.Duration
	broadcastAckThreshold uint
}

// New instantiate a new Broadcaster for a given list of gateway nodes addresses.
// disableBroadcast allows to create a disabled Broadcaster instance.
// connTimeout set the timeout per gRPC connection attempt to a node.
// connAcksThreshold set the lower-bound of required successful gRPC connections to the nodes. If not met, an error will be returned.
// broadcastTimeout set the timeout per proof broadcast.
// broadcastAcksThreshold set the lower-bound of required successful proof broadcasts. If not met, a warning will be logged.
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

	connectionVals := make([]*grpc.ClientConn, len(gatewayAddresses))
	errs := make([]error, len(gatewayAddresses))

	log.Info("Attempting to connect to Spacemesh gateway nodes at %v", gatewayAddresses)
	var wg sync.WaitGroup
	wg.Add(len(gatewayAddresses))
	for i, address := range gatewayAddresses {
		i := i
		address := address
		go func() {
			defer wg.Done()
			connectionVals[i], errs[i] = newClientConn(address, connTimeout)
			if errs[i] == nil {
				log.Info("Successfully connected to Spacemesh gateway node at \"%v\"", address)
			}
		}()
	}
	wg.Wait()

	// Extract successful connections and concatenate errors of non-successful ones.
	var retErr error
	var connections = make([]*grpc.ClientConn, 0)
	for i, err := range errs {
		if err == nil {
			connections = append(connections, connectionVals[i])
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
	if len(connections) < int(connAcksThreshold) {
		for _, conn := range connections {
			_ = conn.Close()

		}
		return nil, retErr
	}

	// If some connections failed, log it.
	if retErr != nil {
		numErrors := len(gatewayAddresses) - len(connections)
		log.Warning("Failed to connect to %d/%d gateway nodes: %v", numErrors, len(gatewayAddresses), retErr)
	}

	clients := make([]pb.GatewayServiceClient, len(connections))
	for i, conn := range connections {
		clients[i] = pb.NewGatewayServiceClient(conn)
	}

	return &Broadcaster{
		clients:               clients,
		connections:           connections,
		broadcastTimeout:      broadcastTimeout,
		broadcastAckThreshold: broadcastAcksThreshold,
	}, nil
}

// BroadcastProof broadcasts a serialized proof of a given round.
func (b *Broadcaster) BroadcastProof(msg []byte, roundID string, members [][]byte) error {
	if b.clients == nil {
		log.Info("Broadcast is disabled, not broadcasting round %v proof", roundID)
		return nil
	}
	pbMsg := &pb.BroadcastPoetRequest{Data: msg}
	ctx, cancel := context.WithTimeout(context.Background(), b.broadcastTimeout)
	defer cancel()

	responses := make([]*pb.BroadcastPoetResponse, len(b.clients))
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
			err = fmt.Errorf("failed to broadcast via gateway node at \"%v\" after %v: %v",
			b.connections[i].Target(), elapsed, err)
		} else if code.Code(res.Status.Code) != code.Code_OK {
			err = fmt.Errorf("failed to broadcast via gateway node at \"%v\" after %v: node response: %s (%s)",
			b.connections[i].Target(), elapsed, code.Code(res.Status.Code).String(), res.Status.GetMessage())
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
		log.Warning("Round %v proof broadcast failed on %d/%d gateway nodes: %v", roundID, numErrors, len(b.clients), retErr)
	}

	log.Info("Round %v proof broadcast completed successfully after %v, num of members: %d, proof size: %d", roundID, elapsed, len(members), len(msg))
	return nil
}

// newClientConn returns a new gRPC client
// connection to the specified target.
func newClientConn(target string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		// XXX: this is done to prevent routers from cleaning up our connections (e.g aws load balances..)
		// TODO: these parameters work for now but we might need to revisit or add them as configuration
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Minute,
			Timeout:             time.Minute * 3,
			PermitWithoutStream: true,
		})}
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rpc server: %v", err)
	}

	return conn, nil
}

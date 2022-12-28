package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/spacemeshos/smutil/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Manager aggregates GRPC connections to gateways.
// Its Close() must be called when the connections are no longer needed.
type Manager struct {
	connections []*grpc.ClientConn
}

func (m *Manager) Connections() []*grpc.ClientConn {
	return m.connections
}

func closeConnections(connections []*grpc.ClientConn) error {
	var result *multierror.Error
	for _, conn := range connections {
		if err := conn.Close(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

func (m *Manager) Close() error {
	return closeConnections(m.connections)
}

// NewManager creates a gateway manager connected to `gateways`.
// Close() must be called when the connections are no longer needed.
func NewManager(ctx context.Context, gateways []string, minSuccesfulConns uint) (*Manager, error) {
	connections, errs := connect(ctx, gateways)
	if len(connections) < int(minSuccesfulConns) {
		errs = multierror.Append(errs, closeConnections(connections))
		return nil, fmt.Errorf("not enough gtw connections (%d/%d): %w", len(connections), minSuccesfulConns, errs)
	}

	return &Manager{connections: connections}, nil
}

// Connect tries to connect to gateways at provided addresses.
// Returns list of connections and errors for connection failures.
func connect(ctx context.Context, gateways []string) ([]*grpc.ClientConn, error) {
	log.Info("Attempting to connect to Spacemesh gateway nodes at %v", gateways)
	connections := make([]*grpc.ClientConn, 0, len(gateways))

	connsChan := make(chan *grpc.ClientConn, len(gateways))

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		// XXX: this is done to prevent routers from cleaning up our connections (e.g aws load balances..)
		// TODO: these parameters work for now but we might need to revisit or add them as configuration
		// Tracked by: https://github.com/spacemeshos/poet/issues/154
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Minute,
			Timeout:             time.Minute * 3,
			PermitWithoutStream: true,
		}),
	}

	var eg multierror.Group
	for _, target := range gateways {
		target := target
		eg.Go(func() error {
			conn, err := grpc.DialContext(ctx, target, opts...)
			if err != nil {
				return fmt.Errorf("failed to connect to gateway grpc server %s (%w)", target, err)
			}
			log.With().Info("Successfully connected to gateway node", log.String("target", conn.Target()))
			connsChan <- conn
			return nil
		})
	}

	err := eg.Wait()
	close(connsChan)

	for conn := range connsChan {
		connections = append(connections, conn)
	}

	return connections, err
}

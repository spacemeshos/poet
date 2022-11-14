package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/spacemeshos/smutil/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const DefaultConnTimeout = 10 * time.Second

type ConnectingErrors struct {
	errors []error
}

func (e *ConnectingErrors) Error() string {
	// concatenate errors
	err := ""
	for _, e := range e.errors {
		if err == "" {
			err = e.Error()
		} else {
			err = fmt.Sprintf("%v | %v", err, e)
		}
	}
	return err
}

func (e *ConnectingErrors) Is(target error) bool {
	_, ok := target.(*ConnectingErrors)
	return ok
}

// Manager aggregates GRPC connections to gateways.
// Its Close() must be called when the connections are no longer needed.
type Manager struct {
	connections []*grpc.ClientConn
}

func (m *Manager) Connections() []*grpc.ClientConn {
	if m == nil {
		return []*grpc.ClientConn{}
	}
	return m.connections
}

func (m *Manager) Close() {
	if m == nil {
		return
	}
	for _, conn := range m.connections {
		conn.Close()
	}
}

// NewManager creates a gateway manager connected to `gateways`.
// Close() must be called when the connections are no longer needed.
func NewManager(ctx context.Context, gateways []string, minSuccesfullConns uint) (*Manager, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultConnTimeout)
	defer cancel()
	connections, errs := connect(ctx, gateways)
	if len(connections) < int(minSuccesfullConns) {
		for _, conn := range connections {
			conn.Close()
		}
		return nil, &ConnectingErrors{errors: errs}
	}

	return &Manager{
		connections: connections,
	}, nil
}

// Connect tries to connect to gateways at provided addresses.
// Returns list of connections and errors for connection failures.
func connect(ctx context.Context, gateways []string) ([]*grpc.ClientConn, []error) {
	log.Info("Attempting to connect to Spacemesh gateway nodes at %v", gateways)
	connections := make([]*grpc.ClientConn, 0, len(gateways))
	errors := make([]error, 0)

	errorsChan := make(chan error, len(gateways))
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

	eg, _ := errgroup.WithContext(ctx)
	for _, target := range gateways {
		target := target
		eg.Go(func() error {
			conn, err := grpc.DialContext(ctx, target, opts...)
			if err != nil {
				errorsChan <- fmt.Errorf("failed to connect to gateway grpc server %s (%w)", target, err)
			} else {
				log.With().Info("Successfully connected to gateway node", log.String("target", conn.Target()))
				connsChan <- conn
			}
			return nil
		})
	}

	egConsumers, _ := errgroup.WithContext(ctx)
	egConsumers.Go(func() error {
		for err := range errorsChan {
			errors = append(errors, err)
		}
		return nil
	})
	egConsumers.Go(func() error {
		for conn := range connsChan {
			connections = append(connections, conn)
		}
		return nil
	})
	eg.Wait()
	close(errorsChan)
	close(connsChan)
	egConsumers.Wait()

	return connections, errors
}

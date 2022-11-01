package gateway

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Connect tries to connect to gateways at provided addresses.
// Returns list of connections and errors for connection failures.
func Connect(ctx context.Context, gateways []string) ([]*grpc.ClientConn, []error) {
	connections := make([]*grpc.ClientConn, 0, len(gateways))
	errors := make([]error, 0)

	errorsChan := make(chan error, len(gateways))
	connsChan := make(chan *grpc.ClientConn, len(gateways))

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		// XXX: this is done to prevent routers from cleaning up our connections (e.g aws load balances..)
		// TODO: these parameters work for now but we might need to revisit or add them as configuration
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

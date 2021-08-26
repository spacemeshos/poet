package main

import (
	"context"
	"fmt"
	"github.com/spacemeshos/poet/rpc/debug"

	"google.golang.org/grpc"
)

func main() {
	addr := "localhost:50002"
	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to dial grpc: %s", err)
	}

	c := debug.NewDebugClient(conn)
	res, err := c.Shutdown(context.Background(), &debug.ShutdownRequest{})

	if err != nil {
		fmt.Printf("Failed to get account details: %s", err)
	}
	fmt.Sprintln("result: %v", res)
}

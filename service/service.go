package service

import (
	"fmt"
	proxy "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spacemeshos/poet-core-api/pcrpc"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"net/http"
)

const (
	rpcAddr      = "localhost:80"
	webProxyAddr = "localhost:8080"
)

func Start() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize, and register our implementation of the gRPC interface
	// exported by the rpcServer.
	rpcServer := NewRPCServer(internal.NewProver, internal.NewVerifier, shared.NewHashFunc, shared.NewScryptHashFunc)
	grpcServer := grpc.NewServer()
	pcrpc.RegisterPoetCoreProverServer(grpcServer, rpcServer)
	pcrpc.RegisterPoetVerifierServer(grpcServer, rpcServer)

	// Start the gRPC service listening for HTTP/2 connections.
	lis, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v\n", err)
	}
	defer lis.Close()

	go func() {
		fmt.Printf("RPC service listening on %s\n", lis.Addr())
		grpcServer.Serve(lis)
	}()

	// Start the web proxy for the gRPC service.
	mux := proxy.NewServeMux()
	err = pcrpc.RegisterPoetCoreProverHandlerFromEndpoint(ctx, mux, rpcAddr, []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
		fmt.Printf("web proxy: failed to register handler from endpoint: %s\n", err)
		return err
	}

	go func() {
		fmt.Printf("webproxy is listening on %s\n", webProxyAddr)
		http.ListenAndServe(webProxyAddr, mux)
	}()

	// Wait for shutdown signal from either a graceful service stop or from
	// the interrupt handler.
	<-ShutdownChannel()
	return nil
}

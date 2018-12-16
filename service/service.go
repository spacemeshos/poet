package service

import (
	"fmt"
	proxy "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spacemeshos/poet-core-api/pcrpc"
	"github.com/spacemeshos/poet-ref/internal"
	"github.com/spacemeshos/poet-ref/shared"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"log"
	"net"
	"net/http"
)

func Start(rpcPort, wProxyPort uint) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rpcAddr := net.JoinHostPort("0.0.0.0", fmt.Sprint(rpcPort))
	webProxyAddr := net.JoinHostPort("0.0.0.0", fmt.Sprint(wProxyPort))

	// Initialize, and register our implementation of the gRPC interface
	// exported by the rpcServer.
	rpcServer := NewRPCServer(internal.NewProver, internal.NewVerifier, shared.NewHashFunc, shared.NewScryptHashFunc)
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(loggerInterceptor()))

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
		fmt.Printf("webproxy start listening on %s\n", webProxyAddr)
		err := http.ListenAndServe(webProxyAddr, mux)
		if err != nil {
			fmt.Printf("webproxy failed listening: %s\n", err)
		}
	}()

	// Wait for shutdown signal from either a graceful service stop or from
	// the interrupt handler.
	<-ShutdownChannel()
	return nil
}

func loggerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		peer, _ := peer.FromContext(ctx)

		maxDispLen := 50
		reqStr := fmt.Sprintf("%v", req)
		var reqDispStr string
		if len(reqStr) > maxDispLen {
			reqDispStr = reqStr[:maxDispLen] + "..."
		} else {
			reqDispStr = reqStr
		}
		log.Printf("%v: %v %v\n", peer.Addr.String(), info.FullMethod, reqDispStr)

		resp, err := handler(ctx, req)

		if err != nil {
			log.Printf("%v: FAILURE %v %s", peer.Addr.String(), info.FullMethod, err)
		}
		return resp, err
	}
}

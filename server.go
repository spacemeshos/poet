package main

import (
	"fmt"
	proxy "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spacemeshos/poet/broadcaster"
	"github.com/spacemeshos/poet/rpc"
	"github.com/spacemeshos/poet/rpc/api"
	"github.com/spacemeshos/poet/rpccore"
	"github.com/spacemeshos/poet/rpccore/apicore"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"net/http"
	"os"
)

// startServer starts the RPC server.
func startServer() error {
	sig := signal.NewSignal()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize and register the implementation of gRPC interface
	var grpcServer *grpc.Server
	var proxyRegstr []func(context.Context, *proxy.ServeMux, string, []grpc.DialOption) error
	options := []grpc.ServerOption{
		grpc.UnaryInterceptor(loggerInterceptor()),
	}

	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		if err := os.Mkdir(cfg.DataDir, 0700); err != nil {
			return err
		}
	}

	if cfg.CoreServiceMode {
		rpcServer := rpccore.NewRPCServer(sig, cfg.DataDir)
		grpcServer = grpc.NewServer(options...)

		apicore.RegisterPoetCoreProverServer(grpcServer, rpcServer)
		apicore.RegisterPoetVerifierServer(grpcServer, rpcServer)
		proxyRegstr = append(proxyRegstr, apicore.RegisterPoetCoreProverHandlerFromEndpoint)
		proxyRegstr = append(proxyRegstr, apicore.RegisterPoetVerifierHandlerFromEndpoint)
	} else {
		svc, err := service.NewService(sig, cfg.Service, cfg.DataDir)
		if err != nil {
			return err
		}

		go func() {
			proofBroadcaster, err := broadcaster.New(cfg.NodeAddress)
			if err != nil {
				log.Error("could not connect to node: %v", err)
				sig.RequestShutdown()
				return
			}
			svc.Start(proofBroadcaster)
		}()

		rpcServer := rpc.NewRPCServer(svc)
		grpcServer = grpc.NewServer(options...)

		api.RegisterPoetServer(grpcServer, rpcServer)
		proxyRegstr = append(proxyRegstr, api.RegisterPoetHandlerFromEndpoint)
	}

	// Start the gRPC server listening for HTTP/2 connections.
	lis, err := net.Listen(cfg.RPCListener.Network(), cfg.RPCListener.String())
	if err != nil {
		return fmt.Errorf("failed to listen: %v\n", err)
	}
	defer lis.Close()

	go func() {
		log.Info("RPC server listening on %s", lis.Addr())
		grpcServer.Serve(lis)
	}()

	// Start the REST proxy for the gRPC server above.
	mux := proxy.NewServeMux()
	for _, r := range proxyRegstr {
		err := r(ctx, mux, cfg.RPCListener.String(), []grpc.DialOption{grpc.WithInsecure()})
		if err != nil {
			return err
		}
	}

	go func() {
		log.Info("REST proxy start listening on %s", cfg.RESTListener.String())
		err := http.ListenAndServe(cfg.RESTListener.String(), mux)
		log.Error("REST proxy failed listening: %s\n", err)
	}()

	// Wait for shutdown signal from either a graceful server stop or from
	// the interrupt handler.
	<-sig.ShutdownChannel()
	return nil
}

// loggerInterceptor returns UnaryServerInterceptor handler to log all RPC server incoming requests.
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
		log.Debug("%v: %v %v", peer.Addr.String(), info.FullMethod, reqDispStr)

		resp, err := handler(ctx, req)

		if err != nil {
			log.Debug("%v: FAILURE %v %s", peer.Addr.String(), info.FullMethod, err)
		}
		return resp, err
	}
}

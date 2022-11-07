package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	proxy "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/gateway/broadcaster"
	"github.com/spacemeshos/poet/gateway/smesher"
	"github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/release/proto/go/rpccore/apicore"
	"github.com/spacemeshos/poet/rpc"
	"github.com/spacemeshos/poet/rpccore"
	"github.com/spacemeshos/poet/service"
)

// startServer starts the RPC server.
func StartServer(ctx context.Context, cfg *config.Config) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	serverGroup, ctx := errgroup.WithContext(ctx)

	// Initialize and register the implementation of gRPC interface
	var grpcServer *grpc.Server
	var proxyRegstr []func(context.Context, *proxy.ServeMux, string, []grpc.DialOption) error
	options := []grpc.ServerOption{
		grpc.UnaryInterceptor(loggerInterceptor()),
		// XXX: this is done to prevent routers from cleaning up our connections (e.g aws load balances..)
		// TODO: these parameters work for now but we might need to revisit or add them as configuration
		// TODO: Configure maxconns, maxconcurrentcons ..
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     time.Minute * 120,
			MaxConnectionAge:      time.Minute * 180,
			MaxConnectionAgeGrace: time.Minute * 10,
			Time:                  time.Minute,
			Timeout:               time.Minute * 3,
		}),
	}

	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		if err := os.Mkdir(cfg.DataDir, 0o700); err != nil {
			return err
		}
	}

	if cfg.CoreServiceMode {
		rpcServer := rpccore.NewRPCServer(stop, cfg.DataDir)
		grpcServer = grpc.NewServer(options...)

		apicore.RegisterPoetCoreProverServer(grpcServer, rpcServer)
		apicore.RegisterPoetVerifierServer(grpcServer, rpcServer)
		proxyRegstr = append(proxyRegstr, apicore.RegisterPoetCoreProverHandlerFromEndpoint)
		proxyRegstr = append(proxyRegstr, apicore.RegisterPoetVerifierHandlerFromEndpoint)
	} else {
		svc, err := service.NewService(cfg.Service, cfg.DataDir)
		defer svc.Shutdown()
		if err != nil {
			return err
		}
		gtwManager, _ := gateway.NewGatewayManager(ctx, cfg.Service.GatewayAddresses)
		if len(gtwManager.Connections()) > 0 {
			postConfig := smesher.FetchPostConfig(ctx, gtwManager.Connections())
			if postConfig == nil {
				return errors.New("failed to fetch post config from gateways")
			}
			broadcaster, err := broadcaster.New(
				gtwManager.Connections(),
				cfg.Service.DisableBroadcast,
				cfg.Service.ConnAcksThreshold,
				broadcaster.DefaultBroadcastTimeout,
				cfg.Service.BroadcastAcksThreshold,
			)
			if err != nil {
				gtwManager.Close()
				return err
			}
			atxProvider, err := service.CreateAtxProvider(gtwManager.Connections())
			if err != nil {
				gtwManager.Close()
				return fmt.Errorf("failed to create ATX provider: %w", err)
			}

			if err := svc.Start(broadcaster, atxProvider, postConfig); err != nil {
				return err
			}
		} else {
			log.Info("Service not starting, waiting for start request")
		}

		rpcServer := rpc.NewRPCServer(svc, gtwManager)
		grpcServer = grpc.NewServer(options...)

		api.RegisterPoetServer(grpcServer, rpcServer)
		proxyRegstr = append(proxyRegstr, api.RegisterPoetHandlerFromEndpoint)
	}

	// Start the gRPC server listening for HTTP/2 connections.
	lis, err := net.Listen(cfg.RPCListener.Network(), cfg.RPCListener.String())
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()

	serverGroup.Go(func() error {
		log.Info("RPC server listening on %s", lis.Addr())
		return grpcServer.Serve(lis)
	})

	// Start the REST proxy for the gRPC server above.
	mux := proxy.NewServeMux()
	for _, r := range proxyRegstr {
		err := r(ctx, mux, cfg.RPCListener.String(), []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())})
		if err != nil {
			return err
		}
	}

	server := &http.Server{Addr: cfg.RESTListener.String(), Handler: mux}
	serverGroup.Go(func() error {
		log.Info("REST proxy start listening on %s", cfg.RESTListener.String())
		return server.ListenAndServe()
	})

	// Wait for the server to shut down gracefully
	<-ctx.Done()
	grpcServer.GracefulStop()
	server.Shutdown(ctx)
	return serverGroup.Wait()
}

// loggerInterceptor returns UnaryServerInterceptor handler to log all RPC server incoming requests.
func loggerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		peer, _ := peer.FromContext(ctx)

		if submitReq, ok := req.(*api.SubmitRequest); ok {
			log.Info("%v | %s | %v", info.FullMethod, submitReq.String(), peer.Addr.String())
		} else {
			maxDispLen := 50
			reqStr := fmt.Sprintf("%v", req)

			var reqDispStr string
			if len(reqStr) > maxDispLen {
				reqDispStr = reqStr[:maxDispLen] + "..."
			} else {
				reqDispStr = reqStr
			}
			log.Info("%v | %v | %v", info.FullMethod, reqDispStr, peer.Addr.String())
		}

		resp, err := handler(ctx, req)
		if err != nil {
			log.Info("FAILURE %v | %v | %v", info.FullMethod, err, peer.Addr.String())
		}
		return resp, err
	}
}

package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	proxy "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spacemeshos/smutil/log"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"

	"github.com/spacemeshos/poet/config"
	"github.com/spacemeshos/poet/gateway"
	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/release/proto/go/rpc/api"
	"github.com/spacemeshos/poet/rpc"
	"github.com/spacemeshos/poet/service"
)

type Server struct {
	svc          *service.Service
	cfg          config.Config
	rpcListener  net.Listener
	restListener net.Listener
}

func New(cfg config.Config) (*Server, error) {
	rpcListener, err := net.Listen(cfg.RPCListener.Network(), cfg.RPCListener.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	restListener, err := net.Listen(cfg.RESTListener.Network(), cfg.RESTListener.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		if err := os.Mkdir(cfg.DataDir, 0o700); err != nil {
			return nil, err
		}
	}

	svc, err := service.NewService(cfg.Service, cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create Service: %v", err)
	}

	return &Server{
		svc:          svc,
		cfg:          cfg,
		rpcListener:  rpcListener,
		restListener: restListener,
	}, nil
}

func (s *Server) Close() {
	s.rpcListener.Close()
	s.restListener.Close()
}

func (s *Server) RpcAddr() net.Addr {
	return s.rpcListener.Addr()
}

// startServer starts the RPC server.
func (s *Server) Start(ctx context.Context) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	serverGroup, ctx := errgroup.WithContext(ctx)

	logger := log.AppLog.WithName("StartServer")
	ctx = logging.NewContext(ctx, logger)

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

	proofsDbPath := filepath.Join(s.cfg.DataDir, "proofs")
	proofsDb, err := service.NewProofsDatabase(proofsDbPath, s.svc.ProofsChan())
	if err != nil {
		return fmt.Errorf("failed to create proofs DB: %w", err)
	}
	serverGroup.Go(func() error {
		return proofsDb.Run(ctx)
	})

	serverGroup.Go(func() error {
		return s.svc.Run(ctx)
	})

	gtwConnCtx, cancel := context.WithTimeout(ctx, s.cfg.GtwConnTimeout)
	defer cancel()
	gtwManager, err := gateway.NewManager(gtwConnCtx, s.cfg.Service.GatewayAddresses, s.cfg.Service.ConnAcksThreshold)
	if err == nil {
		verifier, err := service.CreateChallengeVerifier(gtwManager.Connections())
		if err != nil {
			if err := gtwManager.Close(); err != nil {
				logger.With().Warning("failed to close GRPC connections", log.Err(err))
			}
			return fmt.Errorf("failed to create challenge verifier: %w", err)
		}
		s.svc.Start(verifier)
	} else {
		logger.With().Info("Service not starting, waiting for start request", log.Err(err))
		gtwManager = &gateway.Manager{}
	}

	rpcServer := rpc.NewServer(s.svc, proofsDb, gtwManager, s.cfg)
	grpcServer = grpc.NewServer(options...)

	api.RegisterPoetServer(grpcServer, rpcServer)
	proxyRegstr = append(proxyRegstr, api.RegisterPoetHandlerFromEndpoint)

	// Start the gRPC server listening for HTTP/2 connections.
	serverGroup.Go(func() error {
		logger.Info("RPC server listening on %s", s.rpcListener.Addr())
		return grpcServer.Serve(s.rpcListener)
	})

	// Start the REST proxy for the gRPC server above.
	mux := proxy.NewServeMux()
	for _, r := range proxyRegstr {
		err := r(ctx, mux, s.rpcListener.Addr().String(), []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())})
		if err != nil {
			return err
		}
	}

	server := &http.Server{Handler: mux}
	serverGroup.Go(func() error {
		logger.Info("REST proxy starts listening on %s", s.restListener.Addr())
		err := server.Serve(s.restListener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	// Wait for the server to shut down gracefully
	<-ctx.Done()
	grpcServer.GracefulStop()
	server.Shutdown(ctx)
	return serverGroup.Wait()
}

// loggerInterceptor returns UnaryServerInterceptor handler to log all RPC server incoming requests.
func loggerInterceptor() func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		peer, _ := peer.FromContext(ctx)

		logger := log.AppLog.WithName(info.FullMethod).WithFields(log.Field(zap.Stringer("request_id", uuid.New())))
		ctx = logging.NewContext(ctx, logger)

		if msg, ok := req.(fmt.Stringer); ok {
			logger.With().Debug("new GRPC", log.Field(zap.Stringer("from", peer.Addr)), log.Field(zap.Stringer("message", msg)))
		}

		resp, err := handler(ctx, req)
		if err != nil {
			logger.With().Info("FAILURE", log.Err(err))
		}
		return resp, err
	}
}

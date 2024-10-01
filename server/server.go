package server

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	grpcmw "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	proxy "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"

	"github.com/spacemeshos/poet/logging"
	"github.com/spacemeshos/poet/registration"
	api "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
	"github.com/spacemeshos/poet/rpc"
	"github.com/spacemeshos/poet/service"
	"github.com/spacemeshos/poet/transport"
)

type svc interface {
	Run(ctx context.Context) error
}

type Server struct {
	worker svc
	reg    *registration.Registration
	cfg    Config

	configRpcListener net.Listener
	rpcListener       net.Listener
	restListener      net.Listener

	privateKey ed25519.PrivateKey
}

func New(ctx context.Context, cfg Config) (*Server, error) {
	// Resolve the RPC listener
	addr, err := net.ResolveTCPAddr("tcp", cfg.RawRPCListener)
	if err != nil {
		return nil, err
	}
	rpcListener, err := net.Listen(addr.Network(), addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	// Resolve the config RPC listener
	addr, err = net.ResolveTCPAddr("tcp", cfg.ConfigRPCListener)
	if err != nil {
		return nil, err
	}
	configRpcListener, err := net.Listen(addr.Network(), addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	// Resolve the REST listener
	addr, err = net.ResolveTCPAddr("tcp", cfg.RawRESTListener)
	if err != nil {
		return nil, err
	}
	restListener, err := net.Listen(addr.Network(), addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
			return nil, err
		}
	}

	s, err := loadState(ctx, cfg.DataDir, os.Getenv(KeyEnvVar))
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}
	if err := saveState(cfg.DataDir, s); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}
	privateKey := s.PrivKey

	tr := transport.NewInMemory()
	var worker svc
	if cfg.DisableWorker {
		logging.FromContext(ctx).Info("PoSW worker service is disabled")
		worker = service.NewDisabledService(tr)
	} else {
		worker, err = service.New(
			ctx,
			cfg.Genesis.Time(),
			cfg.DataDir,
			tr,
			cfg.Round,
			service.WithConfig(cfg.Service),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Service: %v", err)
		}
	}

	reg, err := registration.New(
		ctx,
		cfg.Genesis.Time(),
		cfg.DbDir,
		tr,
		cfg.Round,
		registration.WithConfig(cfg.Registration),
		registration.WithPrivateKey(privateKey),
	)
	if err != nil {
		return nil, fmt.Errorf("creating registration service: %w", err)
	}

	return &Server{
		worker:     worker,
		reg:        reg,
		cfg:        cfg,
		privateKey: privateKey,

		configRpcListener: configRpcListener,
		rpcListener:       rpcListener,
		restListener:      restListener,
	}, nil
}

func (s *Server) Close() error {
	return s.reg.Close()
}

// GrpcAddr returns the address that server is listening on for GRPC.
func (s *Server) GrpcAddr() net.Addr {
	return s.rpcListener.Addr()
}

// GrpcAddr returns the address that the configuration server is listening on for GRPC.
func (s *Server) ConfigGrpcAddr() net.Addr {
	return s.configRpcListener.Addr()
}

// GrpcRestProxyAddr returns the address that REST-GRPC proxy is listening on.
func (s *Server) GrpcRestProxyAddr() net.Addr {
	return s.restListener.Addr()
}

// Start starts the RPC server.
func (s *Server) Start(ctx context.Context) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	serverGroup, ctx := errgroup.WithContext(ctx)

	logger := logging.FromContext(ctx)

	metrics := grpc_prometheus.NewServerMetrics(
		grpc_prometheus.WithServerHandlingTimeHistogram(
			grpc_prometheus.WithHistogramBuckets(prometheus.ExponentialBuckets(0.001, 2, 16)),
		),
	)

	logger.Info("starting registration service")
	serverGroup.Go(func() error {
		return s.reg.Run(ctx)
	})

	logger.Info("starting PoSW worker service")
	serverGroup.Go(func() error {
		return s.worker.Run(ctx)
	})

	rpcServer := rpc.NewServer(s.reg, s.cfg.Round.PhaseShift, s.cfg.Round.CycleGap)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmw.ChainUnaryServer(
			loggerInterceptor(logger),
			metrics.UnaryServerInterceptor(),
		)),
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
	)

	api.RegisterPoetServiceServer(grpcServer, rpcServer)

	reflection.Register(grpcServer)
	metrics.InitializeMetrics(grpcServer)

	// Start the gRPC server listening for HTTP/2 connections.
	serverGroup.Go(func() error {
		logger.Sugar().Infof("GRPC server listening on %s", s.rpcListener.Addr())
		return grpcServer.Serve(s.rpcListener)
	})

	configGrpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmw.ChainUnaryServer(loggerInterceptor(logger), metrics.UnaryServerInterceptor())),
	)
	api.RegisterConfigurationServiceServer(configGrpcServer, rpcServer)

	reflection.Register(configGrpcServer)
	metrics.InitializeMetrics(configGrpcServer)
	prometheus.Register(metrics)

	// Start the gRPC server listening for HTTP/2 connections.
	serverGroup.Go(func() error {
		logger.Sugar().Infof("Config GRPC server listening on %s", s.configRpcListener.Addr())
		return configGrpcServer.Serve(s.configRpcListener)
	})

	// Start the REST proxy for the gRPC server above.
	mux := proxy.NewServeMux()
	err := api.RegisterPoetServiceHandlerFromEndpoint(
		ctx,
		mux,
		s.rpcListener.Addr().String(),
		[]grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(s.cfg.MaxGrpcRespSize)),
		},
	)
	if err != nil {
		return err
	}

	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second * 5}
	serverGroup.Go(func() error {
		logger.Sugar().Infof("REST proxy starts listening on %s", s.restListener.Addr())
		err := server.Serve(s.restListener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	// Wait for the server to shut down gracefully
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	grpcServer.GracefulStop()
	configGrpcServer.GracefulStop()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Sugar().Errorf("failed to shutdown server: %s", err)
	}
	if err := serverGroup.Wait(); err != nil {
		logger.Sugar().Errorf("error when waiting to shutdown servers: %s", err)
	}
	return nil
}

func (s *Server) PublicKey() ed25519.PublicKey {
	return s.privateKey.Public().(ed25519.PublicKey)
}

// loggerInterceptor returns UnaryServerInterceptor handler to log all RPC server incoming requests.
func loggerInterceptor(
	logger *zap.Logger,
) func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		peer, _ := peer.FromContext(ctx)

		logger := logger.Named(info.FullMethod).With(zap.Stringer("request_id", uuid.New()))
		ctx = logging.NewContext(ctx, logger)

		if msg, ok := req.(fmt.Stringer); ok {
			logger.Debug("new GRPC", zap.Stringer("from", peer.Addr), zap.Stringer("message", msg))
		}

		resp, err := handler(ctx, req)
		if err != nil {
			logger.Info("FAILURE", zap.Error(err))
		}
		return resp, err
	}
}

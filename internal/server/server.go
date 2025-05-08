package server

import (
	"context"
	"net"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/Jonnydevp/subpub/internal/config"
	"github.com/Jonnydevp/subpub/internal/service"
	pb "github.com/Jonnydevp/subpub/proto"
	"github.com/Jonnydevp/subpub/subpub"
)

type Server struct {
	grpcServer *grpc.Server
	config     *config.Config
	logger     *zap.Logger
	bus        subpub.SubPub
	wg         sync.WaitGroup
}

func New(cfg *config.Config, logger *zap.Logger) *Server {
	return &Server{
		config: cfg,
		logger: logger,
		bus:    subpub.NewSubPub(),
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.config.GRPC.Addr)
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.unaryInterceptor()),
		grpc.StreamInterceptor(s.streamInterceptor()),
	)

	pb.RegisterPubSubServer(s.grpcServer, service.New(s.bus, s.logger))

	s.logger.Info("Starting gRPC server", zap.String("addr", s.config.GRPC.Addr))

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Error("gRPC server failed", zap.Error(err))
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	if err := s.bus.Close(ctx); err != nil {
		s.logger.Warn("Failed to gracefully shutdown pubsub", zap.Error(err))
	}

	select {
	case <-stopped:
		s.logger.Info("Server stopped gracefully")
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		return ctx.Err()
	}
}

func (s *Server) unaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		s.logger.Debug("unary request", zap.String("method", info.FullMethod))
		return handler(ctx, req)
	}
}

func (s *Server) streamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		s.logger.Debug("stream request", zap.String("method", info.FullMethod))
		return handler(srv, ss)
	}
}

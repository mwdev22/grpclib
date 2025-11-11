package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// server is a small wrapper around grpc.Server with helpers for startup/shutdown and service registration.
type Server struct {
	grpcServer *grpc.Server
	lis        net.Listener
	opts       Options
	host       string

	mu sync.Mutex
	// registration functions to call before Serve
	registrants []func(*grpc.Server)
}

// constructs a Server with provided options
func NewServer(host string, opts ...Option) *Server {

	o := Options{
		ShutdownTimeout: 30 * time.Second,
	}

	for _, fn := range opts {
		fn(&o)
	}

	var grpcOpts []grpc.ServerOption
	if len(o.UnaryInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainUnaryInterceptor(o.UnaryInterceptors...))
	}
	if len(o.StreamInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainStreamInterceptor(o.StreamInterceptors...))
	}

	if o.Creds != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(o.Creds))
	}

	s := &Server{
		grpcServer: grpc.NewServer(grpcOpts...),
		opts:       o,
		host:       host,
	}

	if o.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	return s
}

// registers a function that receives the underlying *grpc.Server and performs generated registration.
// example: s.RegisterService(func(gs *grpc.Server) { svc.RegistryService(gs, impl) })
func (s *Server) RegisterService(fn func(*grpc.Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registrants = append(s.registrants, fn)
}

func (s *Server) Start(ctx context.Context) (string, error) {
	s.mu.Lock()
	registrants := s.registrants
	s.mu.Unlock()

	lis, err := net.Listen("tcp", s.host)
	if err != nil {
		return "", fmt.Errorf("listen %s: %w", s.host, err)
	}
	s.lis = lis

	// run registration functions
	for _, reg := range registrants {
		reg(s.grpcServer)
	}

	go func() {
		_ = s.grpcServer.Serve(lis)
	}()

	return lis.Addr().String(), nil
}

func (s *Server) Addr() string {
	if s.lis == nil {
		return ""
	}
	return s.lis.Addr().String()
}

// returns the underlying *grpc.Server for advanced use cases.
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

func (s *Server) Stop(ctx context.Context) error {
	if s == nil || s.grpcServer == nil {
		return errors.New("server not running")
	}

	if s.lis == nil {
		return errors.New("server not started")
	}

	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	timeout := s.opts.ShutdownTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		s.grpcServer.Stop()
		return fmt.Errorf("graceful stop timed out after %s", timeout)
	}
}

// returns a grpc.DialOption appropriate for tests/clients against this server.
func (s *Server) DialOptionForClient() (grpc.DialOption, error) {
	if s == nil {
		return nil, errors.New("server is nil")
	}
	if s.opts.Creds != nil {
		return grpc.WithTransportCredentials(s.opts.Creds), nil
	}
	return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
}

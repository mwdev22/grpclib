package grpcserver

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestStartStop(t *testing.T) {
	srv := New(
		":0",
		WithReflection(true),
		WithShutdownTimeout(1*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	dialOpt, err := srv.DialOptionForClient()
	if err != nil {
		t.Fatalf("couldn't get dial option: %v", err)
	}

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, addr, dialOpt, grpc.WithBlock())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	conn.Close()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()

	if err := srv.Stop(stopCtx); err != nil {
		t.Fatalf("error stopping server: %v", err)
	}
}

func TestNew_WithOptions(t *testing.T) {
	dummyUnaryInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}

	dummyStreamInterceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}

	srv := New(
		":0",
		WithReflection(false),
		WithShutdownTimeout(5*time.Second),
		WithUnaryInterceptors(dummyUnaryInterceptor),
		WithStreamInterceptors(dummyStreamInterceptor),
	)

	if srv == nil {
		t.Fatal("New() returned nil")
	}

	if srv.opts.EnableReflection {
		t.Error("reflection should be disabled, but it's enabled")
	}

	if srv.opts.ShutdownTimeout != 5*time.Second {
		t.Errorf("shutdown timeout: want 5s, got %v", srv.opts.ShutdownTimeout)
	}

	if got := len(srv.opts.UnaryInterceptors); got != 1 {
		t.Errorf("wrong number of unary interceptors: want 1, got %d", got)
	}

	if got := len(srv.opts.StreamInterceptors); got != 1 {
		t.Errorf("wrong number of stream interceptors: want 1, got %d", got)
	}
}

func TestWithCreds(t *testing.T) {
	creds := insecure.NewCredentials()
	srv := New(":0", WithCreds(creds))

	if srv.opts.Creds != creds {
		t.Error("credentials weren't set correctly")
	}
}

func TestServer_RegisterService(t *testing.T) {
	srv := New(":0")

	wasCalled := false
	srv.RegisterService(func(gs *grpc.Server) {
		wasCalled = true
	})

	if len(srv.registrants) != 1 {
		t.Fatalf("should have 1 registrant, got %d", len(srv.registrants))
	}

	for _, registerFunc := range srv.registrants {
		registerFunc(srv.GRPCServer())
	}

	if !wasCalled {
		t.Error("registration callback was never invoked")
	}
}

func TestServer_Addr(t *testing.T) {
	srv := New(":0")

	if addr := srv.Addr(); addr != "" {
		t.Errorf("addr should be empty before Start(), got %q", addr)
	}

	ctx := context.Background()
	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("couldn't start server: %v", err)
	}

	if got := srv.Addr(); got != addr {
		t.Errorf("Addr() mismatch: want %s, got %s", addr, got)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Stop(stopCtx)
}

func TestServer_GRPCServer(t *testing.T) {
	srv := New(":0")

	gs := srv.GRPCServer()
	if gs == nil {
		t.Fatal("GRPCServer() returned nil")
	}
}

func TestServer_Stop_NotStarted(t *testing.T) {
	srv := New(":0")

	ctx := context.Background()
	err := srv.Stop(ctx)
	if err == nil {
		t.Error("Stop() should error on a server that wasn't started")
	}
}

func TestServer_Stop_NilServer(t *testing.T) {
	var srv *Server

	ctx := context.Background()
	err := srv.Stop(ctx)
	if err == nil {
		t.Error("calling Stop() on nil server should return an error")
	}
}

func TestServer_DialOptionForClient_NilServer(t *testing.T) {
	var srv *Server

	_, err := srv.DialOptionForClient()
	if err == nil {
		t.Error("DialOptionForClient() should fail for nil server")
	}
}

func TestServer_DialOptionForClient_WithCreds(t *testing.T) {
	creds := insecure.NewCredentials()
	srv := New(":0", WithCreds(creds))

	dialOpt, err := srv.DialOptionForClient()
	if err != nil {
		t.Fatalf("DialOptionForClient() failed: %v", err)
	}

	if dialOpt == nil {
		t.Error("got nil dial option, expected a valid option")
	}
}

func TestServer_StopWithTimeout(t *testing.T) {
	srv := New(":0", WithShutdownTimeout(100*time.Millisecond))

	ctx := context.Background()
	_, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	srv.Stop(stopCtx)
}

func TestServer_MultipleRegister(t *testing.T) {
	srv := New(":0")

	callCount := 0
	srv.RegisterService(func(gs *grpc.Server) {
		callCount++
	})
	srv.RegisterService(func(gs *grpc.Server) {
		callCount++
	})

	if got := len(srv.registrants); got != 2 {
		t.Fatalf("want 2 registrants, got %d", got)
	}

	for _, registerFunc := range srv.registrants {
		registerFunc(srv.GRPCServer())
	}

	if callCount != 2 {
		t.Errorf("expected both registrants to be called, got %d calls", callCount)
	}
}

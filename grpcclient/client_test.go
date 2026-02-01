package grpcclient

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func TestClientOptions(t *testing.T) {
	opts := Options{
		DialTimeout: 5 * time.Second,
	}

	WithDialTimeout(10 * time.Second)(&opts)
	WithInsecure()(&opts)

	if opts.DialTimeout != 10*time.Second {
		t.Errorf("expected dial timeout 10s, got %v", opts.DialTimeout)
	}

	if opts.Creds == nil {
		t.Error("expected credentials to be set")
	}
}

func TestWithCredentials(t *testing.T) {
	opts := &Options{}
	creds := insecure.NewCredentials()

	WithCredentials(creds)(opts)

	if opts.Creds != creds {
		t.Error("expected credentials to be set")
	}
}

func TestWithKeepalive(t *testing.T) {
	opts := &Options{}
	expectedTime := 60 * time.Second
	expectedTimeout := 20 * time.Second

	WithKeepalive(expectedTime, expectedTimeout)(opts)

	if opts.KeepAliveTime != expectedTime {
		t.Errorf("expected keepalive time %v, got %v", expectedTime, opts.KeepAliveTime)
	}
	if opts.KeepAliveTimeout != expectedTimeout {
		t.Errorf("expected keepalive timeout %v, got %v", expectedTimeout, opts.KeepAliveTimeout)
	}
}

func TestWithUnaryInterceptors(t *testing.T) {
	opts := &Options{}
	interceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	WithUnaryInterceptors(interceptor)(opts)

	if len(opts.UnaryInterceptors) != 1 {
		t.Errorf("expected 1 unary interceptor, got %d", len(opts.UnaryInterceptors))
	}
}

func TestWithStreamInterceptors(t *testing.T) {
	opts := &Options{}
	interceptor := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(ctx, desc, cc, method, opts...)
	}

	WithStreamInterceptors(interceptor)(opts)

	if len(opts.StreamInterceptors) != 1 {
		t.Errorf("expected 1 stream interceptor, got %d", len(opts.StreamInterceptors))
	}
}

func TestWithMaxMessageSize(t *testing.T) {
	opts := &Options{}
	expectedRecv := 8 * 1024 * 1024
	expectedSend := 8 * 1024 * 1024

	WithMaxMessageSize(expectedRecv, expectedSend)(opts)

	if opts.MaxRecvMsgSize != expectedRecv {
		t.Errorf("expected max recv size %d, got %d", expectedRecv, opts.MaxRecvMsgSize)
	}
	if opts.MaxSendMsgSize != expectedSend {
		t.Errorf("expected max send size %d, got %d", expectedSend, opts.MaxSendMsgSize)
	}
}

func TestWithDefaultTimeout(t *testing.T) {
	opts := &Options{}
	expectedTimeout := 45 * time.Second

	WithDefaultTimeout(expectedTimeout)(opts)

	if opts.DefaultTimeout != expectedTimeout {
		t.Errorf("expected default timeout %v, got %v", expectedTimeout, opts.DefaultTimeout)
	}
}

func TestWithRetry(t *testing.T) {
	opts := &Options{}
	expectedMaxRetries := 5

	WithRetry(expectedMaxRetries)(opts)

	if !opts.EnableRetry {
		t.Error("expected retry to be enabled")
	}
	if opts.MaxRetries != expectedMaxRetries {
		t.Errorf("expected max retries %d, got %d", expectedMaxRetries, opts.MaxRetries)
	}
}

func TestWithDialOptions(t *testing.T) {
	opts := &Options{}
	dialOpt := grpc.WithBlock()

	WithDialOptions(dialOpt)(opts)

	if len(opts.DialOptions) != 1 {
		t.Errorf("expected 1 dial option, got %d", len(opts.DialOptions))
	}
}

func TestNewClient_EmptyAddr(t *testing.T) {
	ctx := context.Background()
	_, err := New(ctx, "", WithInsecure())

	if err == nil {
		t.Error("expected error for empty address")
	}
}

func TestNewClient_WithDefaults(t *testing.T) {
	t.Skip("requires network access")

	ctx := context.Background()
	client, err := New(ctx, "localhost:9999", WithInsecure())

	if err == nil {
		defer client.Close()
	}
}

func TestClient_Conn(t *testing.T) {
	client := &Client{
		conn: nil,
	}

	conn := client.Conn()
	if conn != nil {
		t.Error("expected nil connection")
	}
}

func TestClient_Close(t *testing.T) {
	client := &Client{
		conn: nil,
	}

	err := client.Close()
	if err != nil {
		t.Errorf("expected no error closing nil connection, got %v", err)
	}
}

func TestClient_State(t *testing.T) {
	client := &Client{
		conn: nil,
	}

	state := client.State()
	if state != connectivity.Shutdown {
		t.Errorf("expected Shutdown state, got %v", state)
	}
}

func TestClient_IsHealthy(t *testing.T) {
	client := &Client{
		conn: nil,
	}

	if client.IsHealthy() {
		t.Error("expected client with nil connection to be unhealthy")
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	t.Skip("requires running gRPC server")

	ctx := context.Background()
	client, err := New(
		ctx,
		"localhost:50051",
		WithInsecure(),
		WithDialTimeout(5*time.Second),
		WithDefaultTimeout(30*time.Second),
	)

	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if client.Conn() == nil {
		t.Error("expected connection to be non-nil")
	}
}

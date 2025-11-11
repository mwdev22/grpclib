package grpcclient

import (
	"context"
	"testing"
	"time"
)

func TestClientOptions(t *testing.T) {
	opts := Options{
		DialTimeout: 5 * time.Second,
	}

	WithTarget("localhost:50051")(&opts)
	WithDialTimeout(10 * time.Second)(&opts)
	WithInsecure()(&opts)

	if opts.Target != "localhost:50051" {
		t.Errorf("expected target 'localhost:50051', got %s", opts.Target)
	}

	if opts.DialTimeout != 10*time.Second {
		t.Errorf("expected dial timeout 10s, got %v", opts.DialTimeout)
	}

	if opts.Creds == nil {
		t.Error("expected credentials to be set")
	}
}

func TestNewClient_NoTarget(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx)

	if err == nil {
		t.Fatal("expected error for missing target, got nil")
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	// Skip if no server is running
	t.Skip("requires running gRPC server")

	ctx := context.Background()
	client, err := NewClient(
		ctx,
		WithTarget("localhost:50051"),
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

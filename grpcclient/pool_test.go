package grpcclient

import (
	"context"
	"testing"
)

func TestPool(t *testing.T) {
	t.Skip("requires running gRPC server")
	ctx := context.Background()
	pool, err := NewPool(
		ctx,
		3,
		WithTarget("localhost:50051"),
		WithInsecure(),
	)

	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	if pool.Size() != 3 {
		t.Errorf("expected pool size 3, got %d", pool.Size())
	}

	c1 := pool.Get()
	c2 := pool.Get()
	c3 := pool.Get()
	c4 := pool.Get()

	if c1 == nil || c2 == nil || c3 == nil || c4 == nil {
		t.Error("expected all Get() calls to return clients")
	}

	if c1 != c4 {
		t.Error("expected round-robin to wrap around")
	}
}

func TestPoolHealthCheck(t *testing.T) {
	pool := &Pool{
		clients: []*Client{
			{conn: nil},
			{conn: nil},
		},
	}

	if pool.Size() != 2 {
		t.Errorf("expected pool size 2, got %d", pool.Size())
	}

	healthy := pool.HealthyCount()
	if healthy != 0 {
		t.Errorf("expected 0 healthy connections, got %d", healthy)
	}
}

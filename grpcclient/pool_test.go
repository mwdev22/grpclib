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
		"localhost:50051",
		WithInsecure(),
	)

	if err != nil {
		t.Fatalf("familed to create pool: %v", err)
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

func TestNewPool_ZeroSize(t *testing.T) {
	t.Skip("requires network access")

	ctx := context.Background()
	pool, err := NewPool(ctx, 0, "localhost:9999", WithInsecure())

	if err == nil {
		defer pool.Close()
		if pool.Size() != 1 {
			t.Errorf("expected pool size to default to 1, got %d", pool.Size())
		}
	}
}

func TestNewPool_NegativeSize(t *testing.T) {
	t.Skip("requires network access")

	ctx := context.Background()
	pool, err := NewPool(ctx, -5, "localhost:9999", WithInsecure())

	if err == nil {
		defer pool.Close()
		if pool.Size() != 1 {
			t.Errorf("expected pool size to default to 1, got %d", pool.Size())
		}
	}
}

func TestPool_GetConn(t *testing.T) {
	pool := &Pool{
		clients: []*Client{
			{conn: nil},
		},
	}

	conn := pool.GetConn()
	if conn != nil {
		t.Error("expected nil connection from GetConn")
	}
}

func TestPool_GetConn_EmptyPool(t *testing.T) {
	pool := &Pool{
		clients: []*Client{},
	}

	conn := pool.GetConn()
	if conn != nil {
		t.Error("expected nil connection from empty pool")
	}
}

func TestPool_Get_EmptyPool(t *testing.T) {
	pool := &Pool{
		clients: []*Client{},
	}

	client := pool.Get()
	if client != nil {
		t.Error("expected nil client from empty pool")
	}
}

func TestPool_GetHealthy_NoHealthyClients(t *testing.T) {
	pool := &Pool{
		clients: []*Client{
			{conn: nil},
			{conn: nil},
		},
	}

	client := pool.GetHealthy()
	if client != nil {
		t.Error("expected nil when no healthy clients available")
	}
}

func TestPool_Close_EmptyPool(t *testing.T) {
	pool := &Pool{
		clients: []*Client{},
	}

	err := pool.Close()
	if err != nil {
		t.Errorf("expected no error closing empty pool, got %v", err)
	}
}

func TestPool_Close_NilClients(t *testing.T) {
	pool := &Pool{
		clients: []*Client{
			{conn: nil},
			{conn: nil},
		},
	}

	err := pool.Close()
	if err != nil {
		t.Errorf("expected no error closing pool with nil connections, got %v", err)
	}
}

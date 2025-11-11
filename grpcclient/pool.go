package grpcclient

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
)

// pool of client connections for scaling
type Pool struct {
	clients []*Client
	current uint64
	mu      sync.RWMutex
}

type PoolOptions struct {
	Size          int
	ClientOptions []Option
}

func NewPool(ctx context.Context, size int, opts ...Option) (*Pool, error) {
	if size <= 0 {
		size = 1
	}

	pool := &Pool{
		clients: make([]*Client, 0, size),
	}

	for i := 0; i < size; i++ {
		client, err := NewClient(ctx, opts...)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create client %d: %w", i, err)
		}
		pool.clients = append(pool.clients, client)
	}

	return pool, nil
}

// select a client using round-robin
func (p *Pool) Get() *Client {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.clients) == 0 {
		return nil
	}

	idx := atomic.AddUint64(&p.current, 1) % uint64(len(p.clients))
	return p.clients[idx]
}

func (p *Pool) GetConn() *grpc.ClientConn {
	client := p.Get()
	if client == nil {
		return nil
	}
	return client.Conn()
}

func (p *Pool) GetHealthy() *Client {
	p.mu.RLock()
	defer p.mu.RUnlock()

	start := atomic.LoadUint64(&p.current) % uint64(len(p.clients))

	for i := 0; i < len(p.clients); i++ {
		idx := (start + uint64(i)) % uint64(len(p.clients))
		if p.clients[idx].IsHealthy() {
			atomic.StoreUint64(&p.current, idx)
			return p.clients[idx]
		}
	}

	return nil
}

// returns the number of connections in the pool
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.clients)
}

func (p *Pool) HealthyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, client := range p.clients {
		if client.IsHealthy() {
			count++
		}
	}
	return count
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for i, client := range p.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("client %d: %w", i, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close some clients: %v", errs)
	}

	return nil
}

func (p *Pool) WaitForReady(ctx context.Context) error {
	p.mu.RLock()
	clients := make([]*Client, len(p.clients))
	copy(clients, p.clients)
	p.mu.RUnlock()

	if len(clients) == 0 {
		return fmt.Errorf("pool is empty")
	}

	errChan := make(chan error, len(clients))
	successChan := make(chan struct{}, 1)

	for _, client := range clients {
		go func(c *Client) {
			if err := c.WaitForReady(ctx); err != nil {
				errChan <- err
			} else {
				select {
				case successChan <- struct{}{}:
				default:
				}
			}
		}(client)
	}

	select {
	case <-successChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

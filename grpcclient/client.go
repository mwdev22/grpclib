package grpcclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type Client struct {
	conn *grpc.ClientConn
	opts Options
}

func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	options := Options{
		DialTimeout:      5 * time.Second,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		MaxRecvMsgSize:   4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:   4 * 1024 * 1024, // 4MB
		DefaultTimeout:   30 * time.Second,
		MaxRetries:       3,
	}

	for _, opt := range opts {
		opt(&options)
	}

	if options.Target == "" {
		return nil, fmt.Errorf("target address is required")
	}

	var dialOpts []grpc.DialOption

	if options.Creds != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(options.Creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                options.KeepAliveTime,
		Timeout:             options.KeepAliveTimeout,
		PermitWithoutStream: true,
	}))

	dialOpts = append(dialOpts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(options.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(options.MaxSendMsgSize),
		),
	)

	if len(options.UnaryInterceptors) > 0 {
		dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(options.UnaryInterceptors...))
	}
	if len(options.StreamInterceptors) > 0 {
		dialOpts = append(dialOpts, grpc.WithChainStreamInterceptor(options.StreamInterceptors...))
	}

	dialOpts = append(dialOpts, options.DialOptions...)

	dialCtx, cancel := context.WithTimeout(ctx, options.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, options.Target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", options.Target, err)
	}

	return &Client{
		conn: conn,
		opts: options,
	}, nil
}

func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) State() connectivity.State {
	if c.conn != nil {
		return c.conn.GetState()
	}
	return connectivity.Shutdown
}

func (c *Client) WaitForReady(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	state := c.conn.GetState()
	if state == connectivity.Ready {
		return nil
	}

	for {
		if !c.conn.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
		state = c.conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if state == connectivity.Shutdown {
			return fmt.Errorf("connection shut down")
		}
	}
}

func (c *Client) IsHealthy() bool {
	if c.conn == nil {
		return false
	}
	state := c.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

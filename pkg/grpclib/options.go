package grpclib

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Option func(*Options)

type Options struct {
	Addr string // listen address, default ":0"

	// if nil, server will use insecure.
	TLS credentials.TransportCredentials

	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor

	ShutdownTimeout time.Duration

	// enable reflection service
	EnableReflection bool
}

func WithAddr(addr string) Option {
	return func(o *Options) { o.Addr = addr }
}

// sets transport credentials.
func WithTLS(creds credentials.TransportCredentials) Option {
	return func(o *Options) { o.TLS = creds }
}

// appends unary interceptors
func WithUnaryInterceptors(i ...grpc.UnaryServerInterceptor) Option {
	return func(o *Options) { o.UnaryInterceptors = append(o.UnaryInterceptors, i...) }
}

// appends stream interceptors
func WithStreamInterceptors(i ...grpc.StreamServerInterceptor) Option {
	return func(o *Options) { o.StreamInterceptors = append(o.StreamInterceptors, i...) }
}

// sets the graceful shutdown timeout.
func WithShutdownTimeout(d time.Duration) Option {
	return func(o *Options) { o.ShutdownTimeout = d }
}

// enables the gRPC reflection service.
func WithReflection(enabled bool) Option {
	return func(o *Options) { o.EnableReflection = enabled }
}

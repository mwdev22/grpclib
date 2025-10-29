package opt

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Option func(*Options)

type Options struct {
	Addr string

	Creds credentials.TransportCredentials

	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor

	ShutdownTimeout time.Duration

	// enable reflection service
	EnableReflection bool
}

// sets transport credentials.
func WithCreds(creds credentials.TransportCredentials) Option {
	return func(o *Options) { o.Creds = creds }
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

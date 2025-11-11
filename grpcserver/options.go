package grpcserver

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Option func(*Options)

type Options struct {
	Creds credentials.TransportCredentials

	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor

	ShutdownTimeout time.Duration

	EnableReflection bool
}

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

func WithShutdownTimeout(d time.Duration) Option {
	return func(o *Options) { o.ShutdownTimeout = d }
}

func WithReflection(enabled bool) Option {
	return func(o *Options) { o.EnableReflection = enabled }
}

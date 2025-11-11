package grpcclient

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Options struct {
	Target string

	Creds credentials.TransportCredentials

	DialTimeout time.Duration

	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration

	UnaryInterceptors []grpc.UnaryClientInterceptor

	StreamInterceptors []grpc.StreamClientInterceptor

	MaxRecvMsgSize int
	MaxSendMsgSize int

	DefaultTimeout time.Duration

	EnableRetry bool
	MaxRetries  int

	DialOptions []grpc.DialOption
}

type Option func(*Options)

func WithTarget(target string) Option {
	return func(o *Options) { o.Target = target }
}

func WithCredentials(creds credentials.TransportCredentials) Option {
	return func(o *Options) { o.Creds = creds }
}

func WithInsecure() Option {
	return func(o *Options) { o.Creds = insecure.NewCredentials() }
}

func WithDialTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.DialTimeout = timeout }
}

func WithKeepalive(time, timeout time.Duration) Option {
	return func(o *Options) {
		o.KeepAliveTime = time
		o.KeepAliveTimeout = timeout
	}
}

func WithUnaryInterceptors(interceptors ...grpc.UnaryClientInterceptor) Option {
	return func(o *Options) { o.UnaryInterceptors = append(o.UnaryInterceptors, interceptors...) }
}

func WithStreamInterceptors(interceptors ...grpc.StreamClientInterceptor) Option {
	return func(o *Options) { o.StreamInterceptors = append(o.StreamInterceptors, interceptors...) }
}

func WithMaxMessageSize(recvSize, sendSize int) Option {
	return func(o *Options) {
		o.MaxRecvMsgSize = recvSize
		o.MaxSendMsgSize = sendSize
	}
}

func WithDefaultTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.DefaultTimeout = timeout }
}

func WithRetry(maxRetries int) Option {
	return func(o *Options) {
		o.EnableRetry = true
		o.MaxRetries = maxRetries
	}
}

func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *Options) { o.DialOptions = append(o.DialOptions, opts...) }
}

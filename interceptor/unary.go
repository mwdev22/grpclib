package interceptor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mwdev22/grpclib/grpcctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Logger interface for custom logging implementations
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// defaultLogger uses standard log package
type defaultLogger struct{}

func (d *defaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

func (d *defaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

func (d *defaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] "+msg, args...)
}

var DefaultLogger Logger = &defaultLogger{}

func Logging(logger Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = DefaultLogger
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()
		resp, err = handler(ctx, req)
		duration := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("ERROR: [%s] - %v <%s>:%v",
				info.FullMethod, duration, st.Code(), err)
		} else {
			logger.Info("OK: [%s] - %v",
				info.FullMethod, duration)
		}

		return resp, err
	}
}

func Recovery(logger Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = DefaultLogger
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered: method=%s panic=%v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}

func RequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		requestID := ""

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("x-request-id"); len(ids) > 0 {
				requestID = ids[0]
			}
		}

		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		ctx = grpcctx.SetRequestID(ctx, requestID)

		return handler(ctx, req)
	}
}

func Timeout(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		done := make(chan struct{})
		var res any
		var handlerErr error

		go func() {
			res, handlerErr = handler(ctx, req)
			close(done)
		}()

		select {
		case <-done:
			return res, handlerErr
		case <-ctx.Done():
			return nil, status.Error(codes.DeadlineExceeded, "request timeout exceeded")
		}
	}
}

type Validator interface {
	Validate() error
}

func Validation() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if v, ok := req.(Validator); ok {
			if err := v.Validate(); err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		return handler(ctx, req)
	}
}

func RealIP() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
				ctx = grpcctx.SetClientIP(ctx, ips[0])
			} else if ips := md.Get("x-real-ip"); len(ips) > 0 {
				ctx = grpcctx.SetClientIP(ctx, ips[0])
			}
		}

		return handler(ctx, req)
	}
}

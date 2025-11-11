package interceptor

import (
	"context"
	"fmt"
	"time"

	"github.com/mwdev22/grpclib/grpcctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// StreamLogging returns a stream server interceptor for logging
func StreamLogging(logger Logger) grpc.StreamServerInterceptor {
	if logger == nil {
		logger = DefaultLogger
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		logger.Info("gRPC stream started: method=%s isClientStream=%v isServerStream=%v",
			info.FullMethod, info.IsClientStream, info.IsServerStream)

		err := handler(srv, ss)

		duration := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("ERROR: [%s] - %v <%s>:%v",
				info.FullMethod, duration, st.Code(), err)
		} else {
			logger.Info("OK: [%s] - %v",
				info.FullMethod, duration)
		}

		return err
	}
}

// StreamRecovery returns a stream server interceptor for panic recovery
func StreamRecovery(logger Logger) grpc.StreamServerInterceptor {
	if logger == nil {
		logger = DefaultLogger
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered in stream: method=%s panic=%v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(srv, ss)
	}
}

func StreamRequestID() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
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

		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

func StreamRealIP() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
				ctx = grpcctx.SetClientIP(ctx, ips[0])
			} else if ips := md.Get("x-real-ip"); len(ips) > 0 {
				ctx = grpcctx.SetClientIP(ctx, ips[0])
			}
		}

		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

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

func ClientLogging(logger Logger) grpc.UnaryClientInterceptor {
	if logger == nil {
		logger = DefaultLogger
	}
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("ERROR: [%s] - %v <%s>:%v",
				method, duration, st.Code(), err)
		} else {
			logger.Info("OK: [%s] - %v",
				method, duration)
		}

		return err
	}
}

func ClientRetry(maxRetries int, retryableErrors []codes.Code) grpc.UnaryClientInterceptor {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if retryableErrors == nil {
		retryableErrors = []codes.Code{codes.Unavailable, codes.DeadlineExceeded}
	}

	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var err error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			err = invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			st, _ := status.FromError(err)
			isRetryable := false
			for _, code := range retryableErrors {
				if st.Code() == code {
					isRetryable = true
					break
				}
			}

			if !isRetryable || attempt == maxRetries {
				return err
			}
		}

		return err
	}
}

func ClientRequestID() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		requestID := grpcctx.RequestID(ctx)
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func ClientTimeout(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func StreamClientLogging(logger Logger) grpc.StreamClientInterceptor {
	if logger == nil {
		logger = DefaultLogger
	}
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		logger.Info("OK: [%s] - %v",
			method, start)

		cs, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("ERROR: [%s] - %v: %v",
				method, err, st.Code())
			return nil, err
		}

		return &loggingClientStream{
			ClientStream: cs,
			method:       method,
			logger:       logger,
			start:        start,
		}, nil
	}
}

type loggingClientStream struct {
	grpc.ClientStream
	method string
	logger Logger
	start  time.Time
}

func (s *loggingClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	duration := time.Since(s.start)

	if err != nil {
		s.logger.Error("ERROR: [%s] - %v: %v",
			s.method, duration, err)
	} else {
		s.logger.Info("OK: [%s] - %v",
			s.method, duration)
	}

	return err
}

func StreamClientRequestID() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		requestID := grpcctx.RequestID(ctx)
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		return streamer(ctx, desc, cc, method, opts...)
	}
}

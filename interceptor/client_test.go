package interceptor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mwdev22/grpclib/grpcctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type mockClientConn struct{}

func (m *mockClientConn) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	return nil
}

func TestClientLogging(t *testing.T) {
	logger := &mockLogger{}
	interceptor := ClientLogging(logger)

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(logger.infos) < 1 {
		t.Errorf("expected at least 1 info log, got %d: %v", len(logger.infos), logger.infos)
	}
}

func TestClientLoggingWithError(t *testing.T) {
	logger := &mockLogger{}
	interceptor := ClientLogging(logger)

	expectedErr := status.Error(codes.Internal, "client error")
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return expectedErr
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if len(logger.errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.errors))
	}
}

func TestClientRetry(t *testing.T) {
	interceptor := ClientRetry(3, []codes.Code{codes.Unavailable})

	attempts := 0
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		attempts++
		if attempts < 3 {
			return status.Error(codes.Unavailable, "service unavailable")
		}
		return nil
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err != nil {
		t.Errorf("expected no error after retry, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClientRetryNonRetryableError(t *testing.T) {
	interceptor := ClientRetry(3, []codes.Code{codes.Unavailable})

	attempts := 0
	expectedErr := status.Error(codes.PermissionDenied, "permission denied")
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		attempts++
		return expectedErr
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for non-retryable error), got %d", attempts)
	}
}

func TestClientRetryMaxAttempts(t *testing.T) {
	interceptor := ClientRetry(2, []codes.Code{codes.Unavailable})

	attempts := 0
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		attempts++
		return status.Error(codes.Unavailable, "always failing")
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts (1 + 2 retries), got %d", attempts)
	}
}

func TestClientRetryContextCancellation(t *testing.T) {
	interceptor := ClientRetry(10, []codes.Code{codes.Unavailable})

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return status.Error(codes.Unavailable, "unavailable")
	}

	err := interceptor(ctx, "/test.Service/Method", "request", "reply", nil, invoker)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	if attempts > 3 {
		t.Errorf("expected at most 3 attempts before context cancellation, got %d", attempts)
	}
}

func TestClientRequestID(t *testing.T) {
	interceptor := ClientRequestID()

	var capturedMD metadata.MD
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			capturedMD = md
		}
		return nil
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	requestIDs := capturedMD.Get("x-request-id")
	if len(requestIDs) == 0 {
		t.Fatal("expected request ID in metadata, got none")
	}

	if requestIDs[0] == "" {
		t.Error("expected non-empty request ID")
	}
}

func TestClientRequestIDFromContext(t *testing.T) {
	interceptor := ClientRequestID()

	expectedRequestID := "existing-request-123"
	ctx := grpcctx.SetRequestID(context.Background(), expectedRequestID)

	var capturedMD metadata.MD
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			capturedMD = md
		}
		return nil
	}

	err := interceptor(ctx, "/test.Service/Method", "request", "reply", nil, invoker)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	requestIDs := capturedMD.Get("x-request-id")
	if len(requestIDs) == 0 {
		t.Fatal("expected request ID in metadata, got none")
	}

	if requestIDs[0] != expectedRequestID {
		t.Errorf("expected request ID %s, got %s", expectedRequestID, requestIDs[0])
	}
}

func TestClientTimeout(t *testing.T) {
	interceptor := ClientTimeout(100 * time.Millisecond)

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}
}

func TestClientTimeoutSuccess(t *testing.T) {
	interceptor := ClientTimeout(200 * time.Millisecond)

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

type mockClientStream struct {
	grpc.ClientStream
	ctx context.Context
}

func (m *mockClientStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *mockClientStream) CloseSend() error {
	return nil
}

func TestStreamClientLogging(t *testing.T) {
	logger := &mockLogger{}
	interceptor := StreamClientLogging(logger)

	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return &mockClientStream{ctx: ctx}, nil
	}

	desc := &grpc.StreamDesc{
		StreamName:    "TestStream",
		ClientStreams: true,
		ServerStreams: true,
	}

	cs, err := interceptor(context.Background(), desc, nil, "/test.Service/StreamMethod", streamer)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cs == nil {
		t.Fatal("expected client stream, got nil")
	}

	if len(logger.infos) != 1 {
		t.Errorf("expected 1 info log (stream start), got %d: %v", len(logger.infos), logger.infos)
	}

	if loggingStream, ok := cs.(*loggingClientStream); ok {
		_ = loggingStream.CloseSend()

		if len(logger.infos) != 2 {
			t.Errorf("expected 2 info logs (start + close), got %d: %v", len(logger.infos), logger.infos)
		}
	}
}

func TestStreamClientLoggingError(t *testing.T) {
	logger := &mockLogger{}
	interceptor := StreamClientLogging(logger)

	expectedErr := status.Error(codes.Internal, "stream creation failed")
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return nil, expectedErr
	}

	desc := &grpc.StreamDesc{
		StreamName: "TestStream",
	}

	cs, err := interceptor(context.Background(), desc, nil, "/test.Service/StreamMethod", streamer)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if cs != nil {
		t.Error("expected nil client stream on error")
	}

	if len(logger.errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.errors))
	}
}

func TestStreamClientRequestID(t *testing.T) {
	interceptor := StreamClientRequestID()

	var capturedMD metadata.MD
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			capturedMD = md
		}
		return &mockClientStream{ctx: ctx}, nil
	}

	desc := &grpc.StreamDesc{
		StreamName: "TestStream",
	}

	_, err := interceptor(context.Background(), desc, nil, "/test.Service/StreamMethod", streamer)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	requestIDs := capturedMD.Get("x-request-id")
	if len(requestIDs) == 0 {
		t.Fatal("expected request ID in metadata, got none")
	}

	if requestIDs[0] == "" {
		t.Error("expected non-empty request ID")
	}
}

func TestStreamClientRequestIDFromContext(t *testing.T) {
	interceptor := StreamClientRequestID()

	expectedRequestID := "stream-request-456"
	ctx := grpcctx.SetRequestID(context.Background(), expectedRequestID)

	var capturedMD metadata.MD
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			capturedMD = md
		}
		return &mockClientStream{ctx: ctx}, nil
	}

	desc := &grpc.StreamDesc{
		StreamName: "TestStream",
	}

	_, err := interceptor(ctx, desc, nil, "/test.Service/StreamMethod", streamer)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	requestIDs := capturedMD.Get("x-request-id")
	if len(requestIDs) == 0 {
		t.Fatal("expected request ID in metadata, got none")
	}

	if requestIDs[0] != expectedRequestID {
		t.Errorf("expected request ID %s, got %s", expectedRequestID, requestIDs[0])
	}
}

func TestLoggingClientStreamCloseSendError(t *testing.T) {
	logger := &mockLogger{}

	mockStream := &mockClientStream{
		ctx: context.Background(),
	}

	type errorStream struct {
		*mockClientStream
	}
	errStream := &errorStream{mockStream}

	loggingStream := &loggingClientStream{
		ClientStream: errStream,
		method:       "/test.Service/Method",
		logger:       logger,
		start:        time.Now(),
	}

	expectedErr := errors.New("close error")

	logger.Error("gRPC client stream close failed: method=%s duration=%v error=%v",
		loggingStream.method, time.Since(loggingStream.start), expectedErr)

	if len(logger.errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.errors))
	}
}

package interceptor

import (
	"context"
	"testing"

	"github.com/mwdev22/grpclib/grpcctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func TestStreamLogging(t *testing.T) {
	logger := &mockLogger{}
	interceptor := StreamLogging(logger)

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.Service/StreamMethod",
		IsClientStream: true,
		IsServerStream: true,
	}

	ss := &mockServerStream{ctx: context.Background()}
	err := interceptor(nil, ss, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(logger.infos) < 2 {
		t.Errorf("expected at least 2 info logs (start + end), got %d: %v", len(logger.infos), logger.infos)
	}
}

func TestStreamLoggingWithError(t *testing.T) {
	logger := &mockLogger{}
	interceptor := StreamLogging(logger)

	expectedErr := status.Error(codes.Internal, "stream error")
	handler := func(srv any, stream grpc.ServerStream) error {
		return expectedErr
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ss := &mockServerStream{ctx: context.Background()}
	err := interceptor(nil, ss, info, handler)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if len(logger.errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.errors))
	}
}

func TestStreamRecovery(t *testing.T) {
	logger := &mockLogger{}
	interceptor := StreamRecovery(logger)

	handler := func(srv any, stream grpc.ServerStream) error {
		panic("test stream panic")
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ss := &mockServerStream{ctx: context.Background()}
	err := interceptor(nil, ss, info, handler)

	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected Internal code, got %v", st.Code())
	}

	if len(logger.errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.errors))
	}
}

func TestStreamRequestID(t *testing.T) {
	interceptor := StreamRequestID()

	var capturedCtx context.Context
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ss := &mockServerStream{ctx: context.Background()}
	err := interceptor(nil, ss, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	requestID := grpcctx.RequestID(capturedCtx)
	if requestID == "" {
		t.Error("expected request ID to be set in stream context")
	}
}

func TestStreamRequestIDFromMetadata(t *testing.T) {
	interceptor := StreamRequestID()

	expectedRequestID := "test-request-123"
	md := metadata.New(map[string]string{"x-request-id": expectedRequestID})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedCtx context.Context
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ss := &mockServerStream{ctx: ctx}
	err := interceptor(nil, ss, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	requestID := grpcctx.RequestID(capturedCtx)
	if requestID != expectedRequestID {
		t.Errorf("expected request ID %s, got %s", expectedRequestID, requestID)
	}
}

func TestStreamRealIP(t *testing.T) {
	interceptor := StreamRealIP()

	expectedIP := "192.168.1.100"
	md := metadata.New(map[string]string{"x-forwarded-for": expectedIP})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedCtx context.Context
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ss := &mockServerStream{ctx: ctx}
	err := interceptor(nil, ss, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	clientIP := grpcctx.ClientIP(capturedCtx)
	if clientIP != expectedIP {
		t.Errorf("expected client IP %s, got %s", expectedIP, clientIP)
	}
}

func TestStreamRealIPFromXRealIP(t *testing.T) {
	interceptor := StreamRealIP()

	expectedIP := "10.0.0.50"
	md := metadata.New(map[string]string{"x-real-ip": expectedIP})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedCtx context.Context
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ss := &mockServerStream{ctx: ctx}
	err := interceptor(nil, ss, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	clientIP := grpcctx.ClientIP(capturedCtx)
	if clientIP != expectedIP {
		t.Errorf("expected client IP %s, got %s", expectedIP, clientIP)
	}
}

func TestWrappedServerStream(t *testing.T) {
	originalCtx := context.Background()
	newCtx := context.WithValue(originalCtx, "test-key", "test-value")

	wrapped := &wrappedServerStream{
		ServerStream: &mockServerStream{ctx: originalCtx},
		ctx:          newCtx,
	}

	// Test that Context() returns the wrapped context
	if wrapped.Context() != newCtx {
		t.Error("expected wrapped context to be returned")
	}

	// Verify the value is accessible
	if val := wrapped.Context().Value("test-key"); val != "test-value" {
		t.Errorf("expected 'test-value', got %v", val)
	}
}

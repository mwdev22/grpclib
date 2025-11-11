package interceptor

import (
	"context"
	"testing"
	"time"

	"github.com/mwdev22/grpclib/grpcctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockLogger struct {
	infos  []string
	errors []string
	warns  []string
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.infos = append(m.infos, msg)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.errors = append(m.errors, msg)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.warns = append(m.warns, msg)
}

func TestLogging(t *testing.T) {
	logger := &mockLogger{}
	interceptor := Logging(logger)

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	resp, err := interceptor(context.Background(), "request", info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp != "response" {
		t.Errorf("expected 'response', got %v", resp)
	}

	if len(logger.infos) != 1 {
		t.Errorf("expected 1 info log, got %d: %v", len(logger.infos), logger.infos)
	}
}

func TestRecovery(t *testing.T) {
	logger := &mockLogger{}
	interceptor := Recovery(logger)

	handler := func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	resp, err := interceptor(context.Background(), "request", info, handler)

	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
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

func TestRequestID(t *testing.T) {
	interceptor := RequestID()

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := interceptor(context.Background(), "request", info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	requestID := grpcctx.RequestID(capturedCtx)
	if requestID == "" {
		t.Error("expected request ID to be set")
	}
}

func TestTimeout(t *testing.T) {
	interceptor := Timeout(100 * time.Millisecond)

	handler := func(ctx context.Context, req any) (any, error) {
		time.Sleep(200 * time.Millisecond)
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := interceptor(context.Background(), "request", info, handler)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded code, got %v", st.Code())
	}
}

type validatableRequest struct {
	value string
}

func (v *validatableRequest) Validate() error {
	if v.value == "" {
		return status.Error(codes.InvalidArgument, "value is required")
	}
	return nil
}

func TestValidation(t *testing.T) {
	interceptor := Validation()

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Test with invalid request
	_, err := interceptor(context.Background(), &validatableRequest{value: ""}, info, handler)

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument code, got %v", st.Code())
	}

	// Test with valid request
	resp, err := interceptor(context.Background(), &validatableRequest{value: "valid"}, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp != "response" {
		t.Errorf("expected 'response', got %v", resp)
	}
}

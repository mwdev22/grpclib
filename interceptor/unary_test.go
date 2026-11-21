package interceptor

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	m.errors = append(m.errors, fmt.Sprintf(msg, args...))
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

func TestRecovery_HidesPanicValue(t *testing.T) {
	tests := []struct {
		name       string
		panicValue any
		stream     bool
	}{
		{name: "unary string panic", panicValue: "db password=hunter2", stream: false},
		{name: "unary error panic", panicValue: errors.New("dial tcp 10.0.0.5:5432"), stream: false},
		{name: "stream string panic", panicValue: "db password=hunter2", stream: true},
		{name: "stream error panic", panicValue: errors.New("dial tcp 10.0.0.5:5432"), stream: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			var err error
			if tt.stream {
				handler := func(srv any, stream grpc.ServerStream) error {
					panic(tt.panicValue)
				}
				info := &grpc.StreamServerInfo{
					FullMethod: "/test.Service/StreamMethod",
				}
				err = StreamRecovery(logger)(nil, &mockServerStream{}, info, handler)
			} else {
				handler := func(ctx context.Context, req any) (any, error) {
					panic(tt.panicValue)
				}
				info := &grpc.UnaryServerInfo{
					FullMethod: "/test.Service/Method",
				}
				_, err = Recovery(logger)(context.Background(), "request", info, handler)
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got %v", err)
			}
			if st.Code() != codes.Internal {
				t.Errorf("expected Internal code, got %v", st.Code())
			}
			if st.Message() != "internal server error" {
				t.Errorf("expected generic message, got %q", st.Message())
			}

			panicStr := fmt.Sprint(tt.panicValue)
			if strings.Contains(err.Error(), panicStr) {
				t.Errorf("client error leaks panic value: %q", err.Error())
			}

			if len(logger.errors) != 1 {
				t.Fatalf("expected 1 error log, got %d", len(logger.errors))
			}
			if !strings.Contains(logger.errors[0], panicStr) {
				t.Errorf("expected log to contain panic value %q, got %q", panicStr, logger.errors[0])
			}
			if !strings.Contains(logger.errors[0], "goroutine") {
				t.Errorf("expected log to contain stack trace, got %q", logger.errors[0])
			}
		})
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

	resp, err := interceptor(context.Background(), &validatableRequest{value: "valid"}, info, handler)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp != "response" {
		t.Errorf("expected 'response', got %v", resp)
	}
}

package grpcctx

import (
	"context"
	"testing"
)

func TestSetRequestID(t *testing.T) {
	ctx := context.Background()
	expectedID := "test-request-123"

	ctx = SetRequestID(ctx, expectedID)
	actualID := RequestID(ctx)

	if actualID != expectedID {
		t.Errorf("expected request ID %q, got %q", expectedID, actualID)
	}
}

func TestRequestID_NotSet(t *testing.T) {
	ctx := context.Background()
	id := RequestID(ctx)

	if id != "" {
		t.Errorf("expected empty request ID, got %q", id)
	}
}

func TestRequestID_WrongType(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, requestIDKey, 12345) // wrong type

	id := RequestID(ctx)
	if id != "" {
		t.Errorf("expected empty request ID for wrong type, got %q", id)
	}
}

func TestSetClientIP(t *testing.T) {
	ctx := context.Background()
	expectedIP := "192.168.1.1"

	ctx = SetClientIP(ctx, expectedIP)
	actualIP := ClientIP(ctx)

	if actualIP != expectedIP {
		t.Errorf("expected client IP %q, got %q", expectedIP, actualIP)
	}
}

func TestClientIP_NotSet(t *testing.T) {
	ctx := context.Background()
	ip := ClientIP(ctx)

	if ip != "" {
		t.Errorf("expected empty client IP, got %q", ip)
	}
}

func TestClientIP_WrongType(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, clientIPKey, 12345) // wrong type

	ip := ClientIP(ctx)
	if ip != "" {
		t.Errorf("expected empty client IP for wrong type, got %q", ip)
	}
}

func TestMultipleContextValues(t *testing.T) {
	ctx := context.Background()
	expectedID := "request-456"
	expectedIP := "10.0.0.1"

	ctx = SetRequestID(ctx, expectedID)
	ctx = SetClientIP(ctx, expectedIP)

	actualID := RequestID(ctx)
	actualIP := ClientIP(ctx)

	if actualID != expectedID {
		t.Errorf("expected request ID %q, got %q", expectedID, actualID)
	}

	if actualIP != expectedIP {
		t.Errorf("expected client IP %q, got %q", expectedIP, actualIP)
	}
}

func TestOverwriteContextValues(t *testing.T) {
	ctx := context.Background()

	ctx = SetRequestID(ctx, "first-id")
	ctx = SetRequestID(ctx, "second-id")

	actualID := RequestID(ctx)
	if actualID != "second-id" {
		t.Errorf("expected request ID %q, got %q", "second-id", actualID)
	}

	ctx = SetClientIP(ctx, "1.1.1.1")
	ctx = SetClientIP(ctx, "2.2.2.2")

	actualIP := ClientIP(ctx)
	if actualIP != "2.2.2.2" {
		t.Errorf("expected client IP %q, got %q", "2.2.2.2", actualIP)
	}
}

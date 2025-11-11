package grpcserver

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestStartStop(t *testing.T) {
	s := NewServer(
		":0",
		WithReflection(true),
		WithShutdownTimeout(1*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr, err := s.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	dialOpt, err := s.DialOptionForClient()
	if err != nil {
		t.Fatalf("DialOptionForClient: %v", err)
	}

	dctx, dcancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer dcancel()
	conn, err := grpc.DialContext(dctx, addr, dialOpt, grpc.WithBlock())
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	_ = conn.Close()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

# grpclib

A comprehensive Go library that simplifies creating gRPC servers and clients with built-in interceptors, connection pooling, and context utilities.

# key features:

- **easy gRPC server setup** - simple wrapper around `grpc.Server` with lifecycle management

- **mutiple service registration** - register multiple services with a single server

- **easy gRPC client setup** - client wrapper with connection management, health checking, and retries

- **connection pool** - round-robin load balancing across multiple connections for high-throughput clients

- **healthchecks** - client and server health checking utilities

- **production-ready interceptors** - eg. logging, recovery, request ID tracking, validation

- **zero configuration** - Sensible defaults with full customization options

# Installation

```bash

go get github.com/mwdev22/grpclib
```

# server example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/mwdev22/grpclib/grpcserver"
    "github.com/mwdev22/grpclib/interceptor"
    pb "your/proto/package"
)

type yourService struct {
    pb.UnimplementedYourServiceServer
}

func (s *yourService) YourMethod(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // your implementation
    return &pb.Response{}, nil
}

func main() {
    // create server with interceptors
    srv := grpcserver.NewServer(
        ":8080",
        grpcserver.WithReflection(true),
        grpcserver.WithShutdownTimeout(10*time.Second),
        grpcserver.WithUnaryInterceptors(
            interceptor.Recovery(nil),
            interceptor.RequestID(),
            interceptor.Logging(nil),
            interceptor.Validation(),
        ),
    )

    // register your service
    srv.RegisterService(func(gs *grpc.Server) {
        pb.RegisterYourServiceServer(gs, &yourService{})
    })

    ctx := context.Background()
    addr, err := srv.Start(ctx)
    if err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }

    log.Printf("Server listening on %s", addr)

    // srv.Stop(ctx)
}
```

# client example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/mwdev22/grpclib/grpcclient"
    "github.com/mwdev22/grpclib/interceptor"
    "google.golang.org/grpc/codes"
    pb "your/proto/package"
)

func main() {
    ctx := context.Background()

    // Create client with interceptors
    client, err := grpcclient.NewClient(
        ctx,
        grpcclient.WithTarget("localhost:8080"),
        grpcclient.WithInsecure(),
        grpcclient.WithDialTimeout(5*time.Second),
        grpcclient.WithUnaryInterceptors(
            interceptor.ClientRequestID(),
            interceptor.ClientLogging(nil),
            interceptor.ClientRetry(3, []codes.Code{codes.Unavailable}),
        ),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    if err := client.WaitForReady(ctx); err != nil {
        log.Fatalf("Connection not ready: %v", err)
    }

    // use the client
    grpcClient := pb.NewYourServiceClient(client.Conn())
    resp, err := grpcClient.YourMethod(ctx, &pb.Request{})
    if err != nil {
        log.Fatalf("RPC failed: %v", err)
    }

    log.Printf("Response: %v", resp)
}
```

# packages

## grpcserver

Server wrapper with lifecycle management and service registration.

### creating a server

```go
srv := grpcserver.NewServer(
    ":8080",  // Address to listen on
    grpcserver.WithReflection(true),
    grpcserver.WithShutdownTimeout(30*time.Second),
    grpcserver.WithUnaryInterceptors(/* ... */),
    grpcserver.WithStreamInterceptors(/* ... */),
)
```

### server options

- `WithReflection(bool)` - Enable gRPC reflection service
- `WithShutdownTimeout(duration)` - Set graceful shutdown timeout (default: 30s)
- `WithUnaryInterceptors(...interceptors)` - Add unary interceptors
- `WithStreamInterceptors(...interceptors)` - Add stream interceptors
- `WithCreds(credentials)` - Set transport credentials (TLS)

### server methods

- `RegisterService(func(*grpc.Server))` - Register a gRPC service
- `Start(ctx) (string, error)` - Start the server, returns listening address
- `Stop(ctx) error` - Gracefully stop the server
- `Addr() string` - Get the listening address
- `GRPCServer() *grpc.Server` - Access underlying grpc.Server
- `DialOptionForClient()` - Get dial option for testing

## grpcclient

client with connection management, health checking, and automatic retries.

### creating a client

```go
client, err := grpcclient.NewClient(
    ctx,
    grpcclient.WithTarget("localhost:8080"),
    grpcclient.WithInsecure(),
    grpcclient.WithDialTimeout(5*time.Second),
    grpcclient.WithKeepalive(30*time.Second, 10*time.Second),
    grpcclient.WithMaxMessageSize(4*1024*1024, 4*1024*1024),
    grpcclient.WithUnaryInterceptors(/* ... */),
)
```

### client options

- `WithTarget(address)` - Set target address (required)
- `WithInsecure()` - Use insecure connection
- `WithCredentials(creds)` - Set transport credentials (TLS)
- `WithDialTimeout(duration)` - Set dial timeout (default: 5s)
- `WithKeepalive(time, timeout)` - Configure keepalive (default: 30s, 10s)
- `WithUnaryInterceptors(...interceptors)` - Add unary interceptors
- `WithStreamInterceptors(...interceptors)` - Add stream interceptors
- `WithMaxMessageSize(recv, send)` - Set max message sizes (default: 4MB)
- `WithDefaultTimeout(duration)` - Set default call timeout
- `WithRetry(maxRetries)` - Enable retry logic
- `WithDialOptions(...opts)` - Add custom dial options

### client methods

- `Conn() *grpc.ClientConn` - Get underlying connection
- `Close() error` - Close the connection
- `State() connectivity.State` - Get connection state
- `WaitForReady(ctx) error` - Wait for connection to be ready
- `IsHealthy() bool` - Check if connection is healthy

### connection pool

round-robin connection pool for scaling.

```go
pool, err := grpcclient.NewPool(
    ctx,
    5,  // Pool size
    grpcclient.WithTarget("localhost:8080"),
    grpcclient.WithInsecure(),
)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

client := pool.Get()
conn := client.Conn()

// get a healthy connection
healthyClient := pool.GetHealthy()

// check pool health
log.Printf("Pool size: %d, Healthy: %d", pool.Size(), pool.HealthyCount())

// wait for at least one connection to be ready
err = pool.WaitForReady(ctx)
```

### pool methods

- `Get() *Client` - Get next client (round-robin)
- `GetConn() *grpc.ClientConn` - Get next connection
- `GetHealthy() *Client` - Get first healthy client
- `Size() int` - Total pool size
- `HealthyCount() int` - Number of healthy connections
- `Close() error` - Close all connections
- `WaitForReady(ctx) error` - Wait for at least one connection

## interceptor

Production-ready interceptors for servers and clients.

### server unary interceptors

```go
import "github.com/mwdev22/grpclib/interceptor"

// panic recovery
interceptor.Recovery(logger)

// request/response logging
interceptor.Logging(logger)

// request ID tracking
interceptor.RequestID()

// request timeout enforcement
interceptor.Timeout(5 * time.Second)

// request validation (implements Validator interface)
interceptor.Validation()

// extract real client IP from headers
interceptor.RealIP()
```

### server stream interceptors

```go
// stream logging
interceptor.StreamLogging(logger)

// stream panic recovery
interceptor.StreamRecovery(logger)

// stream request ID tracking
interceptor.StreamRequestID()

// stream real IP extraction
interceptor.StreamRealIP()
```

### client unary interceptors

```go
// client request logging
interceptor.ClientLogging(logger)

// automatic retry with backoff
interceptor.ClientRetry(3, []codes.Code{codes.Unavailable, codes.DeadlineExceeded})

// request ID propagation
interceptor.ClientRequestID()

interceptor.ClientTimeout(10 * time.Second)
```

# client stream interceptors

```go
interceptor.StreamClientLogging(logger)

interceptor.StreamClientRequestID()
```

## grpcctx

context utilities for request tracking.

```go
import "github.com/mwdev22/grpclib/grpcctx"

// Set and get request ID
ctx = grpcctx.SetRequestID(ctx, "request-123")
requestID := grpcctx.RequestID(ctx)

// Set and get client IP
ctx = grpcctx.SetClientIP(ctx, "192.168.1.1")
clientIP := grpcctx.ClientIP(ctx)
```

# complete example

## server with all features

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/mwdev22/grpclib/grpcclient"
    "github.com/mwdev22/grpclib/grpcctx"
    "github.com/mwdev22/grpclib/grpcserver"
    "github.com/mwdev22/grpclib/interceptor"
    "google.golang.org/grpc"
    pb "your/proto/package"
)

type service struct {
    pb.UnimplementedYourServiceServer
}

func (s *service) YourMethod(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    requestID := grpcctx.RequestID(ctx)
    log.Printf("Handling request %s", requestID)

    clientIP := grpcctx.ClientIP(ctx)
    log.Printf("Client IP: %s", clientIP)

    return &pb.Response{Message: "Hello"}, nil
}

func main() {
    srv := grpcserver.NewServer(
        ":8080",
        grpcserver.WithReflection(true),
        grpcserver.WithShutdownTimeout(10*time.Second),
        grpcserver.WithUnaryInterceptors(
            interceptor.Recovery(nil),
            interceptor.RequestID(),
            interceptor.RealIP(),
            interceptor.Logging(nil),
            interceptor.Validation(),
            interceptor.Timeout(30*time.Second),
        ),
        grpcserver.WithStreamInterceptors(
            interceptor.StreamRecovery(nil),
            interceptor.StreamRequestID(),
            interceptor.StreamRealIP(),
            interceptor.StreamLogging(nil),
        ),
    )

    // register service
    srv.RegisterService(func(gs *grpc.Server) {
        pb.RegisterYourServiceServer(gs, &service{})
    })

    // start server
    ctx := context.Background()
    addr, err := srv.Start(ctx)
    if err != nil {
        log.Fatalf("Failed to start: %v", err)
    }

    log.Printf("Server running on %s", addr)

    // wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    // graceful shutdown
    log.Println("Shutting down...")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := srv.Stop(shutdownCtx); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
    log.Println("Server stopped")
}
```

## client with connection pool

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/mwdev22/grpclib/grpcclient"
    "github.com/mwdev22/grpclib/grpcctx"
    "github.com/mwdev22/grpclib/interceptor"
    "google.golang.org/grpc/codes"
    pb "your/proto/package"
)

func main() {
    ctx := context.Background()

    // create connection pool
    pool, err := grpcclient.NewPool(
        ctx,
        5, // 5 connections
        grpcclient.WithTarget("localhost:8080"),
        grpcclient.WithInsecure(),
        grpcclient.WithDialTimeout(5*time.Second),
        grpcclient.WithUnaryInterceptors(
            interceptor.ClientRequestID(),
            interceptor.ClientLogging(nil),
            interceptor.ClientRetry(3, []codes.Code{
                codes.Unavailable,
                codes.DeadlineExceeded,
            }),
            interceptor.ClientTimeout(10*time.Second),
        ),
    )
    if err != nil {
        log.Fatalf("Failed to create pool: %v", err)
    }
    defer pool.Close()

    // wait for connections
    if err := pool.WaitForReady(ctx); err != nil {
        log.Fatalf("Pool not ready: %v", err)
    }

    log.Printf("Pool ready: %d/%d connections healthy",
        pool.HealthyCount(), pool.Size())

    // make requests
    for i := 0; i < 10; i++ {
        // get a connection from the pool
        client := pool.Get()
        if client == nil {
            log.Println("No client available")
            continue
        }

        // create service client
        grpcClient := pb.NewYourServiceClient(client.Conn())

        // add request ID to context
        reqCtx := grpcctx.SetRequestID(ctx, fmt.Sprintf("req-%d", i))

        // make the call
        resp, err := grpcClient.YourMethod(reqCtx, &pb.Request{})
        if err != nil {
            log.Printf("Request %d failed: %v", i, err)
            continue
        }

        log.Printf("Request %d succeeded: %v", i, resp)
    }
}
```

# Testing

```bash
go test -v ./...
```

# Best Practices

## server

1. **Always use Recovery interceptor** - prevents server crashes from panics
2. **Enable RequestID** - essential for distributed tracing
3. **Set appropriate timeout** - use `Timeout()` interceptor to prevent long-running requests
4. **Use graceful shutdown** - always call `Stop()` to close the connections, with a context deadline

## client

1. **Use connection pooling** - for high-throughput services
2. **Configure retries wisely** - only retry idempotent operations
3. **Set dial timeout** - fail fast if service is down
4. **Implement health checks** - use `WaitForReady()` and `IsHealthy()`
5. **Propagate request ID** - use `ClientRequestID()` interceptor

## interceptors

1. **Order matters** - recovery should be first, logging should be early
2. **Use custom loggers** - pass your logging implementation
3. **Implement validation** - pass your validation logic
4. **Track request IDs** - use throughout the call chain

# License

MIT

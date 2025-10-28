# grpclib

Small helper library to simplify creating and running gRPC servers in Go.

Key features:

- Build a server with options (address, TLS, interceptors, reflection).
- Register generated gRPC services via a small helper.
- Start and gracefully stop the server with a timeout.

Quick example:

```go
srv := grpclib.NewServer(grpclib.WithAddr(":8080"))
srv.RegisterService(func(gs *grpc.Server) {
    pb.RegisterMyServiceServer(gs, myImpl)
})
addr, _ := srv.Start(context.Background())
// ...
srv.Stop(context.Background())
```

See `pkg/grpclib` for the implementation and tests.

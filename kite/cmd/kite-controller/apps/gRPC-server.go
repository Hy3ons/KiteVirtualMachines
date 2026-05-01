package apps

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

const defaultGRPCServerAddress = ":50051"

// RunGRPCServer starts the kite controller gRPC server on the default port.
// ctx is used to stop the server with GracefulStop when the controller shuts down.
// This function is used by cmd/kite-controller/main.go after Kubernetes clients are initialized.
func RunGRPCServer(ctx context.Context) error {
	return RunGRPCServerWithAddress(ctx, defaultGRPCServerAddress)
}

// RunGRPCServerWithAddress starts the kite controller gRPC server on the given TCP address.
// ctx is used to stop the server with GracefulStop when the controller shuts down.
// address is the TCP listen address, for example ":50051".
// This function is used by RunGRPCServer and tests that need a custom listen address.
func RunGRPCServerWithAddress(ctx context.Context, address string) error {
	if address == "" {
		return fmt.Errorf("gRPC address is empty")
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC address %s: %w", address, err)
	}

	server := grpc.NewServer()
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
		case <-done:
		}

		// 두개 중 하나가 종료되기 전까지 nonBlocking 상태.
		server.GracefulStop()
	}()

	defer close(done)

	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server on %s: %w", address, err)
	}

	return nil
}

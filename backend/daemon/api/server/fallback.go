package server

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// RegisterGRPC is a no-op fallback when protobuf stubs are not generated.
func RegisterGRPC(s *grpc.Server, impl *DaemonAPIServer) {}

// RegisterGateway returns an error to trigger native HTTP fallback when stubs are not generated.
func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return fmt.Errorf("gateway not available: protobuf stubs not generated")
}

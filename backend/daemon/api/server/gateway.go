package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	// Fallback to native HTTP handlers if protobuf stubs are not present
)

// StartAPIServers starts the gRPC server, HTTP gateway, and an SSE endpoint.
// grpcAddr: address for gRPC (e.g., 127.0.0.1:9090)
// restAddr: address for REST (e.g., 127.0.0.1:8080)
func StartAPIServers(ctx context.Context, grpcAddr, restAddr string, impl *DaemonAPIServer) (grpcStop func(), restStop func(), err error) {
	// Start gRPC server
	grpcServer := grpc.NewServer()
	// Attempt to register gRPC service if generated stubs exist; otherwise skip
	// RegisterGRPC is a no-op in native HTTP mode
	RegisterGRPC(grpcServer, impl)
	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, nil, err
	}
	go func() { _ = grpcServer.Serve(l) }()
	grpcStop = func() { grpcServer.GracefulStop(); _ = l.Close() }

	// Start HTTP mux; try grpc-gateway else fallback to native handlers
	gwMux := http.NewServeMux()
	// Try to register grpc-gateway on a separate mux and mount under "/"
	gw := runtime.NewServeMux(runtime.WithErrorHandler(JSONErrorHandler))
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := RegisterGateway(ctx, gw, grpcAddr, dialOpts); err == nil {
		gwMux.Handle("/", gw)
	} else {
		// Fallback to native handlers
		impl.RegisterHTTP(gwMux)
	}

	// Compose SSE handler under /api/v1/events
	root := http.NewServeMux()
	root.Handle("/api/v1/events", SSEHandler(impl.events))
	root.Handle("/", gwMux)
	// Optional auth: enforce X-Auth-Token if QUANTARAX_AUTH_TOKEN is set
	authToken := os.Getenv("QUANTARAX_AUTH_TOKEN")
	var handler http.Handler = root
	if authToken != "" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Auth-Token") != authToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			root.ServeHTTP(w, r)
		})
	}
	server := &http.Server{Addr: restAddr, Handler: handler}
	go func() { _ = server.ListenAndServe() }()
	restStop = func() { _ = server.Close() }
	return grpcStop, restStop, nil
}

// JSONErrorHandler converts gateway errors to a normalized JSON model
func JSONErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	st, ok := status.FromError(err)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"INTERNAL","message":"internal error"}`))
		return
	}
	code := codeToString(st.Code())
	httpStatus := runtime.HTTPStatusFromCode(st.Code())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	payload := map[string]interface{}{"code": code, "message": st.Message()}
	b, _ := json.Marshal(payload)
	_, _ = w.Write(b)
}

func codeToString(c codes.Code) string {
	switch c {
	case codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case codes.NotFound:
		return "NOT_FOUND"
	case codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case codes.Unauthenticated:
		return "UNAUTHENTICATED"
	case codes.Unimplemented:
		return "UNIMPLEMENTED"
	case codes.Unavailable:
		return "UNAVAILABLE"
	default:
		return "INTERNAL"
	}
}

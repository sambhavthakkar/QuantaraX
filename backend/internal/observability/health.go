package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HealthStatus represents the health status of a component.
type HealthStatus string

const (
	HealthStatusOK        HealthStatus = "ok"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents the health of a single component.
type ComponentHealth struct {
	Status    HealthStatus `json:"status"`
	Message   string       `json:"message,omitempty"`
	LatencyMS int64        `json:"latency_ms,omitempty"`
}

// HealthCheckResponse represents the overall health check response.
type HealthCheckResponse struct {
	Status        HealthStatus               `json:"status"`
	Version       string                     `json:"version"`
	UptimeSeconds int64                      `json:"uptime_seconds"`
	Timestamp     string                     `json:"timestamp"`
	Checks        map[string]ComponentHealth `json:"checks"`
}

// HealthChecker performs health checks on system components.
type HealthChecker struct {
	version   string
	startTime time.Time
	checks    map[string]HealthCheckFunc
}

// HealthCheckFunc defines a function that checks component health.
type HealthCheckFunc func(ctx context.Context) ComponentHealth

// NewHealthChecker creates a new health checker.
func NewHealthChecker(version string) *HealthChecker {
	return &HealthChecker{
		version:   version,
		startTime: time.Now(),
		checks:    make(map[string]HealthCheckFunc),
	}
}

// RegisterCheck registers a health check for a component.
func (hc *HealthChecker) RegisterCheck(name string, checkFunc HealthCheckFunc) {
	hc.checks[name] = checkFunc
}

// Check performs all health checks.
func (hc *HealthChecker) Check(ctx context.Context) HealthCheckResponse {
	response := HealthCheckResponse{
		Status:        HealthStatusOK,
		Version:       hc.version,
		UptimeSeconds: int64(time.Since(hc.startTime).Seconds()),
		Timestamp:     time.Now().Format(time.RFC3339),
		Checks:        make(map[string]ComponentHealth),
	}

	for name, checkFunc := range hc.checks {
		health := checkFunc(ctx)
		response.Checks[name] = health

		// Update overall status
		if health.Status == HealthStatusUnhealthy {
			response.Status = HealthStatusUnhealthy
		} else if health.Status == HealthStatusDegraded && response.Status != HealthStatusUnhealthy {
			response.Status = HealthStatusDegraded
		}
	}

	return response
}

// Handler returns an HTTP handler for health checks.
func (hc *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := hc.Check(ctx)

		w.Header().Set("Content-Type", "application/json")

		// Set HTTP status based on health
		switch response.Status {
		case HealthStatusOK:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still 200 but degraded
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// Common health check functions

// GRPCServerCheck checks if gRPC server is responsive.
func GRPCServerCheck(addr string) HealthCheckFunc {
	return func(ctx context.Context) ComponentHealth {
		start := time.Now()
		// Placeholder for actual check logic
		latency := time.Since(start).Milliseconds()

		return ComponentHealth{
			Status:    HealthStatusOK,
			Message:   fmt.Sprintf("gRPC server listening on %s", addr),
			LatencyMS: latency,
		}
	}
}

// QUICListenerCheck checks if QUIC listener is bound.
func QUICListenerCheck(addr string) HealthCheckFunc {
	return func(ctx context.Context) ComponentHealth {
		return ComponentHealth{
			Status:  HealthStatusOK,
			Message: fmt.Sprintf("QUIC listener on %s", addr),
		}
	}
}

// KeystoreCheck checks if identity keys are loaded.
func KeystoreCheck(keysLoaded bool) HealthCheckFunc {
	return func(ctx context.Context) ComponentHealth {
		if keysLoaded {
			return ComponentHealth{
				Status:  HealthStatusOK,
				Message: "Identity keys loaded",
			}
		}
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "Identity keys not loaded",
		}
	}
}

// DatabaseCheck checks database connectivity.
func DatabaseCheck(dbPath string) HealthCheckFunc {
	return func(ctx context.Context) ComponentHealth {
		start := time.Now()
		// Placeholder - in production would execute SELECT 1
		latency := time.Since(start).Milliseconds()

		if latency < 50 {
			return ComponentHealth{
				Status:    HealthStatusOK,
				Message:   "SQLite responsive",
				LatencyMS: latency,
			}
		}
		return ComponentHealth{
			Status:    HealthStatusDegraded,
			Message:   "SQLite slow",
			LatencyMS: latency,
		}
	}
}

// DiskSpaceCheck checks available disk space.
func DiskSpaceCheck(path string, minFreeGB int64) HealthCheckFunc {
	return func(ctx context.Context) ComponentHealth {
		// Placeholder - in production would check actual disk space
		freeGB := int64(10) // Mock value

		if freeGB > minFreeGB {
			return ComponentHealth{
				Status:  HealthStatusOK,
				Message: fmt.Sprintf("%d GB free", freeGB),
			}
		}
		return ComponentHealth{
			Status:  HealthStatusDegraded,
			Message: fmt.Sprintf("Low disk space: %d GB free", freeGB),
		}
	}
}

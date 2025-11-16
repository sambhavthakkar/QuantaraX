package transport

import (
	"context"
	"crypto/tls"

	"github.com/quic-go/quic-go"
)

// QUICConnection wraps a QUIC connection with helper methods
type QUICConnection struct {
	conn          *quic.Conn
	controlStream *ControlStream
	scheduler     *PriorityScheduler
}

// NewQUICConnection creates a new QUIC connection wrapper
func NewQUICConnection(conn *quic.Conn) *QUICConnection {
	qc := &QUICConnection{
		conn: conn,
	}
	qc.scheduler = NewPriorityScheduler(conn)
	return qc
}

// OpenControlStream opens the control stream (Stream 0)
func (q *QUICConnection) OpenControlStream(ctx context.Context) (*ControlStream, error) {
	// Stream 0 is always the control stream
	stream, err := q.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}

	q.controlStream = NewControlStream(stream)
	return q.controlStream, nil
}

// AcceptControlStream accepts the control stream from peer
func (q *QUICConnection) AcceptControlStream(ctx context.Context) (*ControlStream, error) {
	stream, err := q.conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}

	q.controlStream = NewControlStream(stream)
	return q.controlStream, nil
}

// GetControlStream returns the existing control stream
func (q *QUICConnection) GetControlStream() *ControlStream {
	return q.controlStream
}

// GetConnection returns the underlying QUIC connection
func (q *QUICConnection) GetConnection() *quic.Conn {
	return q.conn
}

// Scheduler returns the connection's priority scheduler
func (q *QUICConnection) Scheduler() *PriorityScheduler {
	return q.scheduler
}

// Close closes the QUIC connection
func (q *QUICConnection) Close() error {
	if q.controlStream != nil {
		q.controlStream.Close()
	}
	if q.scheduler != nil {
		q.scheduler.Close()
	}
	return q.conn.CloseWithError(0, "connection closed")
}

// DialQUIC establishes a QUIC connection to a remote address
func DialQUIC(ctx context.Context, addr string, tlsConfig *tls.Config) (*QUICConnection, error) {
	conn, err := quic.DialAddr(ctx, addr, tlsConfig, &quic.Config{
		KeepAlivePeriod:                10 * 1e9, // 10s
		MaxIdleTimeout:                 60 * 1e9,
		InitialStreamReceiveWindow:     8 << 20,   // 8 MiB
		InitialConnectionReceiveWindow: 128 << 20, // 128 MiB
	})
	if err != nil {
		return nil, err
	}

	return NewQUICConnection(conn), nil
}

// ListenQUIC starts a QUIC listener
func ListenQUIC(addr string, tlsConfig *tls.Config) (*QUICListener, error) {
	listener, err := quic.ListenAddr(addr, tlsConfig, &quic.Config{
		KeepAlivePeriod:                10 * 1e9,
		MaxIdleTimeout:                 60 * 1e9,
		InitialStreamReceiveWindow:     8 << 20,
		InitialConnectionReceiveWindow: 128 << 20,
	})
	if err != nil {
		return nil, err
	}

	return &QUICListener{listener: listener}, nil
}

// QUICListener wraps a QUIC listener
type QUICListener struct {
	listener *quic.Listener
}

// Accept accepts a new QUIC connection
func (l *QUICListener) Accept(ctx context.Context) (*QUICConnection, error) {
	conn, err := l.listener.Accept(ctx)
	if err != nil {
		return nil, err
	}

	return NewQUICConnection(conn), nil
}

// Close closes the listener
func (l *QUICListener) Close() error {
	return l.listener.Close()
}

// Addr returns the listener's network address
func (l *QUICListener) Addr() string {
	return l.listener.Addr().String()
}

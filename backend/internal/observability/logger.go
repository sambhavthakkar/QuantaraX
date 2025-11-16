package observability

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog for structured logging.
type Logger struct {
	logger zerolog.Logger
}

// NewLogger creates a new structured logger.
func NewLogger(service, version string, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	zerolog.TimeFieldFormat = time.RFC3339

	logger := zerolog.New(output).With().
		Timestamp().
		Str("service", service).
		Str("version", version).
		Str("host", getHostname()).
		Logger()

	return &Logger{
		logger: logger,
	}
}

// WithSession adds session_id context to logger.
func (l *Logger) WithSession(sessionID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("session_id", sessionID).Logger(),
	}
}

// WithPeer adds peer_id context to logger.
func (l *Logger) WithPeer(peerID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("peer_id", peerID).Logger(),
	}
}

// WithFile adds file context to logger.
func (l *Logger) WithFile(filePath string, fileSize int64) *Logger {
	return &Logger{
		logger: l.logger.With().
			Str("file_path", filePath).
			Int64("file_size", fileSize).
			Logger(),
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Info logs an info message.
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Error logs an error message.
func (l *Logger) Error(err error, msg string) {
	l.logger.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(err error, msg string) {
	l.logger.Fatal().Err(err).Msg(msg)
}

// TransferStarted logs transfer start event.
func (l *Logger) TransferStarted(sessionID, filePath string, fileSize int64, totalChunks int) {
	l.logger.Info().
		Str("session_id", sessionID).
		Str("file_path", filePath).
		Int64("file_size", fileSize).
		Int("total_chunks", totalChunks).
		Msg("transfer session started")
}

// ChunkSent logs chunk send event.
func (l *Logger) ChunkSent(sessionID string, chunkIndex int, chunkSize int, streamID int64) {
	l.logger.Debug().
		Str("session_id", sessionID).
		Int("chunk_index", chunkIndex).
		Int("chunk_size", chunkSize).
		Int64("stream_id", streamID).
		Msg("chunk sent on stream")
}

// TransferProgress logs transfer progress.
func (l *Logger) TransferProgress(sessionID string, chunksSent, totalChunks int, transferRate int64, elapsed time.Duration) {
	progress := float64(chunksSent) / float64(totalChunks) * 100.0

	l.logger.Info().
		Str("session_id", sessionID).
		Int("chunks_sent", chunksSent).
		Int("total_chunks", totalChunks).
		Float64("progress_percent", progress).
		Int64("transfer_rate", transferRate).
		Float64("elapsed_seconds", elapsed.Seconds()).
		Msg("transfer progress")
}

// TransferCompleted logs transfer completion.
func (l *Logger) TransferCompleted(sessionID string, fileSize int64, totalChunks int, duration time.Duration, avgThroughput int64, merkleVerified bool) {
	l.logger.Info().
		Str("session_id", sessionID).
		Int64("file_size", fileSize).
		Int("total_chunks", totalChunks).
		Float64("duration_seconds", duration.Seconds()).
		Int64("average_throughput", avgThroughput).
		Bool("merkle_verified", merkleVerified).
		Msg("transfer completed successfully")
}

// ChunkDecryptFailed logs chunk decryption failure.
func (l *Logger) ChunkDecryptFailed(sessionID string, chunkIndex int, errorCode string, errorMsg string, retryCount int) {
	l.logger.Error().
		Str("session_id", sessionID).
		Int("chunk_index", chunkIndex).
		Str("error_code", errorCode).
		Str("error_message", errorMsg).
		Int("retry_count", retryCount).
		Msg("chunk decryption failed")
}

// ConnectionEstablished logs connection establishment.
func (l *Logger) ConnectionEstablished(remoteAddr string, connectionID string) {
	l.logger.Info().
		Str("remote_addr", remoteAddr).
		Str("connection_id", connectionID).
		Msg("QUIC connection established")
}

// ConnectionFailed logs connection failure.
func (l *Logger) ConnectionFailed(remoteAddr string, err error) {
	l.logger.Error().
		Str("remote_addr", remoteAddr).
		Err(err).
		Msg("QUIC connection failed")
}

// Helper function to get hostname.
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

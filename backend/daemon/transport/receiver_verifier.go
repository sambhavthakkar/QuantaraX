package transport

import (
	"errors"
)

// ZeroLossVerifier enforces strict completion policy for Medical domain.
// Placeholder: tracks missing chunks and requires all present before completion.
type ZeroLossVerifier struct {
	total   int
	recvd   map[int]bool
}

func NewZeroLossVerifier(totalChunks int) *ZeroLossVerifier {
	return &ZeroLossVerifier{total: totalChunks, recvd: make(map[int]bool)}
}

func (z *ZeroLossVerifier) MarkReceived(idx int) {
	z.recvd[idx] = true
}

func (z *ZeroLossVerifier) VerifyComplete() error {
	if len(z.recvd) != z.total {
		return errors.New("verification failed: missing chunks in strict mode")
	}
	return nil
}

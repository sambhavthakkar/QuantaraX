package crypto

import (
	"encoding/binary"
)

// DeriveNonce generates a deterministic 12-byte nonce from the IVBase and a counter.
//
// GCM mode requires a unique nonce for every encryption operation under the same key.
// This function ensures nonce uniqueness by XORing the IVBase (derived per session)
// with an encoded counter (e.g., chunk index or message counter).
//
// The nonce derivation formula:
//   Nonce = IVBase XOR (counter encoded as 8-byte little-endian, padded to 12 bytes)
//
// Parameters:
//   - ivBase: 12-byte base initialization vector from session keys
//   - counter: Monotonically increasing counter (chunk index or message counter)
//
// Returns:
//   - 12-byte nonce suitable for AES-GCM
//
// Security Properties:
//   - Each unique counter value produces a unique nonce
//   - Deterministic: same counter always produces same nonce (for given IVBase)
//   - No nonce reuse possible as long as counter doesn't repeat in a session
func DeriveNonce(ivBase [12]byte, counter uint64) [12]byte {
	var nonce [12]byte

	// Encode counter as 8-byte little-endian
	var counterBytes [8]byte
	binary.LittleEndian.PutUint64(counterBytes[:], counter)

	// XOR the first 8 bytes of IVBase with the counter
	for i := 0; i < 8; i++ {
		nonce[i] = ivBase[i] ^ counterBytes[i]
	}

	// Copy the remaining 4 bytes of IVBase unchanged
	copy(nonce[8:12], ivBase[8:12])

	return nonce
}

// DeriveChunkNonce is a convenience wrapper for deriving nonces for chunk encryption.
// It uses the chunk index as the counter.
//
// Parameters:
//   - ivBase: 12-byte base initialization vector from session keys
//   - chunkIndex: Zero-based chunk index
//
// Returns:
//   - 12-byte nonce for encrypting this chunk
func DeriveChunkNonce(ivBase [12]byte, chunkIndex uint32) [12]byte {
	return DeriveNonce(ivBase, uint64(chunkIndex))
}

// DeriveControlNonce is a convenience wrapper for deriving nonces for control messages.
// It uses the message counter as the counter, offset by a large value to avoid
// collision with chunk nonces.
//
// Parameters:
//   - ivBase: 12-byte base initialization vector from session keys
//   - messageCounter: Monotonically increasing message counter
//
// Returns:
//   - 12-byte nonce for encrypting this control message
func DeriveControlNonce(ivBase [12]byte, messageCounter uint32) [12]byte {
	// Offset control message counters to avoid collision with chunk indices
	// Use high bit to distinguish: 0x8000000000000000 | messageCounter
	const controlOffset = uint64(1) << 63
	return DeriveNonce(ivBase, controlOffset|uint64(messageCounter))
}
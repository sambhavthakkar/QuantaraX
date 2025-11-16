package transport

// TODO(merkle): The end-of-transfer verification currently computes the Merkle root from manifest chunk hashes.
// In a full implementation, we should compute the root from the actual received file bytes (or chunk CAS entries)
// to ensure local disk write integrity, and verify signature using service identity keys.

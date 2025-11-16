package fec

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
)

// Encoder handles Reed-Solomon FEC encoding
type Encoder struct {
	k int // data shards
	r int // parity shards
	rs reedsolomon.Encoder
}

// NewEncoder creates a new FEC encoder
func NewEncoder(k, r int) (*Encoder, error) {
	if k < 1 || k > 256 {
		return nil, fmt.Errorf("data shards must be between 1 and 256, got %d", k)
	}
	if r < 1 || r > 256 {
		return nil, fmt.Errorf("parity shards must be between 1 and 256, got %d", r)
	}

	rs, err := reedsolomon.New(k, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create reed-solomon encoder: %w", err)
	}

	return &Encoder{
		k:  k,
		r:  r,
		rs: rs,
	}, nil
}

// Encode generates parity shards from data shards
func (e *Encoder) Encode(dataShards [][]byte) ([][]byte, error) {
	if len(dataShards) != e.k {
		return nil, fmt.Errorf("expected %d data shards, got %d", e.k, len(dataShards))
	}

	// Validate all shards have same size
	if len(dataShards) > 0 {
		shardSize := len(dataShards[0])
		for i, shard := range dataShards {
			if len(shard) != shardSize {
				return nil, fmt.Errorf("shard %d size mismatch: expected %d, got %d", i, shardSize, len(shard))
			}
		}
	}

	// Create parity shards
	parityShards := make([][]byte, e.r)
	for i := range parityShards {
		if len(dataShards) > 0 {
			parityShards[i] = make([]byte, len(dataShards[0]))
		}
	}

	// Combine data and parity shards for encoding
	allShards := make([][]byte, e.k+e.r)
	copy(allShards[:e.k], dataShards)
	copy(allShards[e.k:], parityShards)

	// Encode
	if err := e.rs.Encode(allShards); err != nil {
		return nil, fmt.Errorf("encoding failed: %w", err)
	}

	// Return only the parity shards
	return allShards[e.k:], nil
}

// GetParameters returns the K and R values
func (e *Encoder) GetParameters() (k, r int) {
	return e.k, e.r
}

// Decoder handles Reed-Solomon FEC decoding
type Decoder struct {
	k  int // data shards
	r  int // parity shards
	rs reedsolomon.Encoder
}

// NewDecoder creates a new FEC decoder
func NewDecoder(k, r int) (*Decoder, error) {
	if k < 1 || k > 256 {
		return nil, fmt.Errorf("data shards must be between 1 and 256, got %d", k)
	}
	if r < 1 || r > 256 {
		return nil, fmt.Errorf("parity shards must be between 1 and 256, got %d", r)
	}

	rs, err := reedsolomon.New(k, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create reed-solomon decoder: %w", err)
	}

	return &Decoder{
		k:  k,
		r:  r,
		rs: rs,
	}, nil
}

// Reconstruct reconstructs missing shards in place
func (d *Decoder) Reconstruct(shards [][]byte) error {
	if len(shards) != d.k+d.r {
		return fmt.Errorf("expected %d shards (k=%d + r=%d), got %d", d.k+d.r, d.k, d.r, len(shards))
	}

	// Count missing shards
	missing := 0
	for _, shard := range shards {
		if shard == nil {
			missing++
		}
	}

	if missing > d.r {
		return fmt.Errorf("too many missing shards: %d missing, can only recover up to %d", missing, d.r)
	}

	if missing == 0 {
		return nil // Nothing to reconstruct
	}

	// Reconstruct missing shards
	if err := d.rs.Reconstruct(shards); err != nil {
		return fmt.Errorf("reconstruction failed: %w", err)
	}

	return nil
}

// GetParameters returns the K and R values
func (d *Decoder) GetParameters() (k, r int) {
	return d.k, d.r
}
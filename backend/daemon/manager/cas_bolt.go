package manager

import (
	"path/filepath"
	"time"
	"encoding/binary"
	"github.com/boltdb/bolt"
)

type BoltCAS struct { db *bolt.DB }

// GCRule defines a naive retention rule: delete entries older than MaxAge or when limit exceeded.
// For this simple key-only CAS, we drop keys older than MaxAge.
// In a real CAS, values could include size and path to prune disk usage.

var bucketCAS = []byte("cas")

func OpenBoltCAS(path string) (*BoltCAS, error) {
	db, err := bolt.Open(filepath.Clean(path), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error { _, e := tx.CreateBucketIfNotExists(bucketCAS); return e })
	if err != nil { db.Close(); return nil, err }
	return &BoltCAS{db: db}, nil
}

func (b *BoltCAS) Close() error { return b.db.Close() }

func (b *BoltCAS) HasChunk(hash string) bool {
	var ok bool
	_ = b.db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket(bucketCAS)
		if bk == nil { return nil }
		v := bk.Get([]byte(hash))
		ok = v != nil
		return nil
	})
	return ok
}

func (b *BoltCAS) PutChunk(hash string, length int) error {
	// Store with a simple timestamp value for GC
	return b.db.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket(bucketCAS)
		if bk == nil { return bolt.ErrBucketNotFound }
		// value is 8-byte unix seconds
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(time.Now().Unix()))
		return bk.Put([]byte(hash), buf)
	})
}

// GC removes entries older than maxAge.
func (b *BoltCAS) GC(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge).Unix()
	removed := 0
	err := b.db.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket(bucketCAS)
		if bk == nil { return bolt.ErrBucketNotFound }
		c := bk.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if len(v) >= 8 {
				ts := int64(binary.BigEndian.Uint64(v))
				if ts < cutoff {
					if err := c.Delete(); err != nil { return err }
					removed++
				}
			}
		}
		return nil
	})
	return removed, err
}

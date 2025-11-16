package service

import (
	"time"
	"strconv"
	"github.com/boltdb/bolt"
)

type DTNItem struct {
	SessionID string
	ChunkIdx  int64
	Priority  int
	ExpireAt  int64
}

type DTNQueue struct { db *bolt.DB }

var bucketDTN = []byte("dtn_queue")

func OpenDTNQueue(path string) (*DTNQueue, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error { _, e := tx.CreateBucketIfNotExists(bucketDTN); return e })
	if err != nil { db.Close(); return nil, err }
	return &DTNQueue{db: db}, nil
}

func (q *DTNQueue) Enqueue(item *DTNItem) error {
	return q.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketDTN)
		key := []byte(item.SessionID + ":" + strconv.FormatInt(item.ChunkIdx, 10))
		val := []byte{byte(item.Priority)}
		return b.Put(key, val)
	})
}

func (q *DTNQueue) DequeueBatch(n int) ([]DTNItem, error) {
	var out []DTNItem
	err := q.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketDTN)
		c := b.Cursor()
		for k, v := c.First(); k != nil && len(out) < n; k, v = c.Next() {
			// simplistic parse: find ':'
			var sess string; var idx int64
			for i := range k {
				if k[i] == ':' { sess = string(k[:i]); break }
			}
			// naive: parse idx from suffix
			var mul int64 = 1
			for i := len(k)-1; i >= 0; i-- { if k[i] == ':' { break }; idx += int64(k[i]-'0') * mul; mul *= 10 }
			out = append(out, DTNItem{SessionID: sess, ChunkIdx: idx, Priority: int(v[0])})
			_ = b.Delete(k)
		}
		return nil
	})
	return out, err
}

func (q *DTNQueue) Close() error { return q.db.Close() }

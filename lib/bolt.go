package lib

import (
	"time"

	"github.com/boltdb/bolt"
)

type BoltStorage struct {
	db *bolt.DB
}

func OpenBoltStorage(path string, bucket []byte) (*BoltStorage, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 30 * time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &BoltStorage{db}, nil
}

func (s *BoltStorage) WALName() string {
	return s.db.Path()
}

func (s *BoltStorage) Set(bucket []byte, k []byte, v []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucket).Put(k, v)
	})
}

func (s *BoltStorage) Get(bucket []byte, k []byte) (b []byte, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b = tx.Bucket(bucket).Get(k)
		return nil
	})
	return
}

func (s *BoltStorage) Delete(bucket []byte, k []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucket).Delete(k)
	})
}

func (s *BoltStorage) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if err := fn(k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *BoltStorage) Close() error {
	return s.db.Close()
}

package jumpi

import (
	"errors"

	"github.com/boltdb/bolt"
)

type Store struct {
	db *bolt.DB
}

var (
	BucketSecrets = []string{"secrets"}
	BucketTargets = []string{"targets"}

	ErrNoBucketGiven = errors.New("no bucket specified")
)

func traverseBuckets(bucket []string, tx *bolt.Tx) (*bolt.Bucket, error) {
	if len(bucket) == 0 {
		return nil, ErrNoBucketGiven
	}

	b := tx.Bucket([]byte(bucket[0]))
	if b == nil {
		return nil, errors.New("bucket '" + bucket[0] + "' does not exist")
	}

	if len(bucket) > 1 {
		for nb := range bucket[1:] {
			b = b.Bucket([]byte(bucket[nb]))
			if b == nil {
				return nil, errors.New("bucket '" + bucket[0] + "' does not exist")
			}
		}
	}

	return b, nil
}

func NewStore(filename string) (*Store, error) {
	db, err := bolt.Open(filename, 0600, nil)
	if err != nil {
		return nil, err
	}
	store := &Store{
		db: db,
	}

	// create all needed buckets
	if err := store.Create(BucketSecrets); err != nil {
		return nil, err
	}
	if err := store.Create(BucketTargets); err != nil {
		return nil, err
	}

	return store, nil
}

func (store *Store) Close() {
	store.db.Close()
}

func (store *Store) Set(bucket []string, key, value string) error {
	err := store.db.Update(func(tx *bolt.Tx) error {
		b, err := traverseBuckets(bucket, tx)
		if err != nil {
			return nil
		}

		err = b.Put([]byte(key), []byte(value))
		return err
	})
	return err
}

func (store *Store) Get(bucket []string, key string) (string, error) {
	var value string
	err := store.db.View(func(tx *bolt.Tx) error {
		b, err := traverseBuckets(bucket, tx)
		if err != nil {
			return nil
		}

		value = string(b.Get([]byte(key)))
		return nil
	})

	return value, err
}

func (store *Store) Delete(bucket []string, key string) error {
	err := store.db.View(func(tx *bolt.Tx) error {
		b, err := traverseBuckets(bucket, tx)
		if err != nil {
			return nil
		}

		err = b.Delete([]byte(key))
		return err
	})

	return err
}

func (store *Store) Create(bucket []string) error {
	err := store.db.Update(func(tx *bolt.Tx) error {
		if len(bucket) == 0 {
			return ErrNoBucketGiven
		}

		b, err := tx.CreateBucketIfNotExists([]byte(bucket[0]))
		if err != nil {
			return err
		}

		if len(bucket) > 1 {
			for nb := range bucket[1:] {
				_, err := b.CreateBucketIfNotExists([]byte(bucket[nb]))
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	return err
}

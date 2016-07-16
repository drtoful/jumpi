package jumpi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"hash"

	"github.com/boltdb/bolt"
	"github.com/drtoful/jumpi/utils"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

type Store struct {
	db       *bolt.DB
	locked   bool
	password []byte
}

const (
	HashSHA256 int = 0
)

var (
	DerivationIterations = 8192
)

type metaKeyDerivation struct {
	Salt       string `json:"salt"`
	Iterations int    `json:"iter"`
	Type       int    `json:"type"`
	Challenge  string `json:"challenge"`
}

var (
	BucketMeta        = []string{"meta"}
	BucketSecrets     = []string{"secrets"}
	BucketSecretsKeys = []string{"secrets", "keys"}
	BucketTargets     = []string{"targets"}
	BucketSessions    = []string{"sessions"}

	ErrNoBucketGiven = errors.New("no bucket specified")
	ErrLocked        = errors.New("store is locked")
	ErrUnknownHash   = errors.New("unknown hash algorithm for key derivation")
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
		db:     db,
		locked: true,
	}

	// create all needed buckets
	if err := store.Create(BucketMeta); err != nil {
		return nil, err
	}
	if err := store.Create(BucketSecrets); err != nil {
		return nil, err
	}
	if err := store.Create(BucketSecretsKeys); err != nil {
		return nil, err
	}
	if err := store.Create(BucketTargets); err != nil {
		return nil, err
	}
	if err := store.Create(BucketSessions); err != nil {
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
	err := store.db.Update(func(tx *bolt.Tx) error {
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

func (store *Store) Password() ([]byte, error) {
	if store.locked {
		return nil, ErrLocked
	}

	return store.password, nil
}

func (store *Store) IsLocked() bool {
	return store.locked
}

func (store *Store) Lock() error {
	if _, err := rand.Read(store.password); err != nil {
		return err
	}
	store.locked = true
	return nil
}

func (store *Store) Unlock(password string) error {
	data, err := store.Get(BucketMeta, "keyderivation")
	meta := &metaKeyDerivation{}
	if err != nil {
		return err
	}

	// create new meta information, if this is our first unlock
	if len(data) == 0 {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return err
		}

		challenge, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		if err != nil {
			return err
		}

		meta = &metaKeyDerivation{
			Salt:       utils.Hexlify(salt),
			Iterations: DerivationIterations,
			Type:       HashSHA256,
			Challenge:  string(challenge),
		}
		jdata, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		err = store.Set(BucketMeta, "keyderivation", string(jdata))
		if err != nil {
			return err
		}
	} else {
		if err := json.Unmarshal([]byte(data), meta); err != nil {
			return err
		}

		if err := bcrypt.CompareHashAndPassword([]byte(meta.Challenge), []byte(password)); err != nil {
			return err
		}
	}

	salt, err := utils.UnHexlify(meta.Salt)
	if err != nil {
		return err
	}

	var digest func() hash.Hash
	switch meta.Type {
	case HashSHA256:
		digest = sha256.New
	default:
		return ErrUnknownHash
	}
	store.password = pbkdf2.Key([]byte(password), salt, meta.Iterations, 32, digest)
	store.locked = false

	return nil
}

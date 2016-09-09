package jumpi

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"log"
	"os"

	"github.com/boltdb/bolt"
	"github.com/codahale/chacha20"
	"github.com/drtoful/jumpi/utils"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/ssh/terminal"
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

type keyvalue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type recordKey struct {
	Rounds int    `json:"rounds"`
	Nonce  string `json:"nonce"`
	Data   string `json:"data"`
}

type record struct {
	Key    recordKey `json:"key"`
	Type   string    `json:"type"`
	Rounds int       `json:"rounds"`
	Nonce  string    `json:"nonce"`
	Data   string    `json:"data"`
}

var (
	BucketMeta       = []string{"meta"}
	BucketMetaAdmins = []string{"meta", "admins"}
	BucketSecrets    = []string{"secrets"}
	BucketTargets    = []string{"targets"}
	BucketSessions   = []string{"sessions"}
	BucketUsers      = []string{"users"}
	BucketRoles      = []string{"roles"}
	BucketCasts      = []string{"casts"}

	ErrNoBucketGiven     = errors.New("no bucket specified")
	ErrLocked            = errors.New("store is locked")
	ErrUnknownHash       = errors.New("unknown hash algorithm for key derivation")
	ErrUnsupportedCipher = errors.New("unknown cipher for store")

	DefaultRounds int = 20
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
	if err := store.Create(BucketMetaAdmins); err != nil {
		return nil, err
	}
	if err := store.Create(BucketSecrets); err != nil {
		return nil, err
	}
	if err := store.Create(BucketTargets); err != nil {
		return nil, err
	}
	if err := store.Create(BucketSessions); err != nil {
		return nil, err
	}
	if err := store.Create(BucketUsers); err != nil {
		return nil, err
	}
	if err := store.Create(BucketRoles); err != nil {
		return nil, err
	}
	if err := store.Create(BucketCasts); err != nil {
		return nil, err
	}

	return store, nil
}

func (store *Store) Close() {
	store.db.Close()
}

func (store *Store) encrypt(data []byte) ([]byte, error) {
	if store.locked {
		return nil, ErrLocked
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	defer func() {
		rand.Read(key)
	}()

	nonceKey := make([]byte, 8)
	if _, err := rand.Read(nonceKey); err != nil {
		return nil, err
	}

	nonceData := make([]byte, 8)
	if _, err := rand.Read(nonceData); err != nil {
		return nil, err
	}

	stream, err := chacha20.NewWithRounds(key, nonceData, uint8(DefaultRounds))
	if err != nil {
		return nil, err
	}
	stream.XORKeyStream(data, data)

	stream, err = chacha20.NewWithRounds(store.password, nonceKey, uint8(DefaultRounds))
	if err != nil {
		return nil, err
	}
	stream.XORKeyStream(key, key)

	rkey := recordKey{
		Rounds: DefaultRounds,
		Nonce:  utils.Hexlify(nonceKey),
		Data:   utils.Hexlify(key),
	}
	r := &record{
		Key:    rkey,
		Type:   "chacha20",
		Rounds: DefaultRounds,
		Nonce:  utils.Hexlify(nonceData),
		Data:   utils.Hexlify(data),
	}

	return json.Marshal(r)
}

func (store *Store) decrypt(data []byte) ([]byte, error) {
	if store.locked {
		return nil, ErrLocked
	}

	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}

	// currently only chacha20 encryption is supported
	if r.Type != "chacha20" {
		return nil, ErrUnsupportedCipher
	}

	nonce, err := utils.UnHexlify(r.Key.Nonce)
	if err != nil {
		return nil, err
	}

	key, err := utils.UnHexlify(r.Key.Data)
	if err != nil {
		return nil, err
	}
	defer func() {
		rand.Read(key)
	}()

	stream, err := chacha20.NewWithRounds(store.password, nonce, uint8(r.Key.Rounds))
	if err != nil {
		return nil, err
	}
	stream.XORKeyStream(key, key)

	nonce, err = utils.UnHexlify(r.Nonce)
	if err != nil {
		return nil, err
	}

	rdata, err := utils.UnHexlify(r.Data)
	if err != nil {
		return nil, err
	}

	stream, err = chacha20.NewWithRounds(key, nonce, uint8(r.Rounds))
	if err != nil {
		return nil, err
	}
	stream.XORKeyStream(rdata, rdata)
	return rdata, nil
}

func (store *Store) SetRaw(bucket []string, key string, value []byte) error {
	err := store.db.Update(func(tx *bolt.Tx) error {
		b, err := traverseBuckets(bucket, tx)
		if err != nil {
			return err
		}

		return b.Put([]byte(key), value)
	})
	return err
}

func (store *Store) Set(bucket []string, key string, value []byte) error {
	data, err := store.encrypt(value)
	if err != nil {
		return err
	}
	defer func() {
		rand.Read(data)
	}()

	return store.SetRaw(bucket, key, data)
}

func (store *Store) GetRaw(bucket []string, key string) ([]byte, error) {
	var value []byte
	err := store.db.View(func(tx *bolt.Tx) error {
		b, err := traverseBuckets(bucket, tx)
		if err != nil {
			return err
		}

		value = b.Get([]byte(key))
		return nil
	})

	return value, err
}

func (store *Store) Get(bucket []string, key string) ([]byte, error) {
	data, err := store.GetRaw(bucket, key)
	if err != nil {
		return nil, err
	}

	return store.decrypt(data)
}

func (store *Store) Scan(bucket []string, q string, skip, limit int) ([]*keyvalue, error) {
	result := make([]*keyvalue, 0)
	err := store.db.View(func(tx *bolt.Tx) error {
		b, err := traverseBuckets(bucket, tx)
		if err != nil {
			return nil
		}

		prefix := []byte(q)
		c := b.Cursor()
		n := -1
		for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			n += 1
			if n < skip {
				continue
			}
			if n == skip+limit && limit > 0 {
				break
			}
			if len(k) == 0 {
				break
			}

			// check if key is a bucket
			nb := b.Bucket(k)
			if nb != nil {
				n -= 1
				continue
			}

			v := string(b.Get([]byte(k)))
			result = append(result, &keyvalue{Key: string(k), Value: string(v)})
		}

		return nil
	})

	return result, err
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

func (store *Store) Unlock(password []byte) error {
	data, err := store.GetRaw(BucketMeta, "keyderivation")
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

		challenge, err := bcrypt.GenerateFromPassword(password, 12)
		if err != nil {
			return err
		}
		defer func() {
			rand.Read(challenge)
		}()

		meta = &metaKeyDerivation{
			Salt:       utils.Hexlify(salt),
			Iterations: DerivationIterations,
			Type:       HashSHA256,
			Challenge:  utils.Hexlify(challenge),
		}
		jdata, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		err = store.SetRaw(BucketMeta, "keyderivation", jdata)
		if err != nil {
			return err
		}
	} else {
		if err := json.Unmarshal(data, meta); err != nil {
			return err
		}
		challenge, err := utils.UnHexlify(meta.Challenge)
		if err != nil {
			return err
		}
		defer func() {
			rand.Read(challenge)
		}()

		if err := bcrypt.CompareHashAndPassword(challenge, password); err != nil {
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
	store.password = pbkdf2.Key(password, salt, meta.Iterations, 32, digest)
	defer func() {
		rand.Read(password)
	}()
	store.locked = false

	return nil
}

func readPwd(msg string) ([]byte, error) {
	fmt.Printf(msg)
	pwd1, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	fmt.Printf("Repeat: ")
	pwd2, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	if bytes.Compare(pwd1, pwd2) != 0 {
		return nil, errors.New("inavlid repeated password")
	}

	return pwd1, nil
}

func (store *Store) Auth(username string, password []byte) bool {
	hash, err := store.GetRaw(BucketMetaAdmins, username)
	if err != nil {
		return false
	}
	defer func() {
		rand.Read(password)
	}()

	err = bcrypt.CompareHashAndPassword(hash, password)
	if err != nil {
		return false
	}
	return true
}

func (store *Store) FTR() {
	fmt.Println("Looks like this is the first time that you are running this database")

	// store password
	pwd, err := readPwd("Enter Unlock Password: ")
	if err != nil {
		log.Fatalf("ftr failed: %s\n", err.Error())
	}

	if err := store.Unlock(pwd); err != nil {
		log.Fatalf("ftr failed: %s\n", err.Error())
	}
	defer func() {
		store.Lock()
	}()

	// admin password (for ui/api)
	pwd, err = readPwd("Enter Admin Password: ")
	if err != nil {
		log.Fatalf("ftr failed: %s\n", err.Error())
	}
	defer func() {
		rand.Read(pwd)
	}()

	challenge, err := bcrypt.GenerateFromPassword(pwd, 12)
	if err != nil {
		log.Fatalf("ftr failed: %s\n", err.Error())
	}
	if err := store.SetRaw(BucketMetaAdmins, "admin", challenge); err != nil {
		log.Fatalf("ftr failed: %s\n", err.Error())
	}

	fmt.Println("Setup complete")
}

package jumpi

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/drtoful/jumpi/utils"
)

var (
	ErrWrongKeyFormat = errors.New("wrong publickey format")
	ErrUnknownUser    = errors.New("user unknown")
)

type User struct {
	Name           string
	KeyFingerprint string
}

func UserFromPublicKey(name string, publickey string) (*User, error) {
	// convert public key of user to sha256 fingerprint
	// for later use
	splits := strings.Split(publickey, " ")
	if len(splits) == 1 {
		return nil, ErrWrongKeyFormat
	}

	data, err := base64.StdEncoding.DecodeString(splits[1])
	if err != nil {
		return nil, err
	}

	digest := sha256.New()
	digest.Write(data)
	id := utils.Hexlify(digest.Sum(nil))

	return &User{Name: name, KeyFingerprint: id}, nil
}

func (user *User) Store(store *Store) error {
	return store.Set(BucketUsers, user.KeyFingerprint, []byte(user.Name))
}

func (user *User) Load(store *Store) error {
	name, err := store.Get(BucketUsers, user.KeyFingerprint)
	if err != nil {
		return err
	}

	if len(name) == 0 {
		return ErrUnknownUser
	}

	user.Name = string(name)
	return nil
}

func (user *User) Delete(store *Store) error {
	return store.Delete(BucketUsers, user.KeyFingerprint)
}

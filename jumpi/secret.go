package jumpi

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"

	"github.com/drtoful/jumpi/utils"
	"golang.org/x/crypto/ssh"
)

type TypeSecret int

const (
	Password TypeSecret = 0
	PKey     TypeSecret = 1
)

var (
	ErrUnknownSecretType = errors.New("unknown secret type")
)

type entry struct {
	Data string `json:"data"`
	Type int    `json:"type"`
}

type Secret struct {
	ID     string
	Type   TypeSecret
	Secret interface{}
}

func (secret *Secret) Fingerprint() string {
	if secret.Type == PKey {
		if val, ok := secret.Secret.(*rsa.PrivateKey); ok {
			public := val.Public()
			sshpublic, err := ssh.NewPublicKey(public)
			if err == nil {
				return ssh.FingerprintSHA256(sshpublic)
			}
		}
	}

	return ""
}

func (secret *Secret) Load(store *Store) error {
	var e entry

	jdata, err := store.Get(BucketSecrets, secret.ID)
	if err != nil {
		return err
	}
	defer func() {
		rand.Read(jdata)
	}()

	if err := json.Unmarshal(jdata, &e); err != nil {
		return err
	}
	s, err := utils.UnHexlify(e.Data)
	if err != nil {
		return err
	}

	// try to convert recovered data, to the type
	// of the secret
	secret.Type = TypeSecret(e.Type)
	switch secret.Type {
	case Password:
		secret.Secret = string(s)
	case PKey:
		pkey, err := x509.ParsePKCS1PrivateKey(s)
		if err != nil {
			return err
		}
		secret.Secret = pkey
	default:
		return ErrUnknownSecretType
	}

	return nil
}

func (secret *Secret) Store(store *Store) error {
	// convert secret (whatever it is) to a byte stream, ready
	// to be encrypted
	var data []byte = nil
	switch s := secret.Secret.(type) {
	case string:
		data = []byte(s)
		secret.Type = Password
	case *rsa.PrivateKey:
		data = x509.MarshalPKCS1PrivateKey(s)
		secret.Type = PKey
	default:
		return ErrUnknownSecretType
	}
	defer func() {
		rand.Read(data)
	}()

	if data == nil {
		return ErrUnknownSecretType
	}

	e := &entry{
		Data: utils.Hexlify(data),
		Type: int(secret.Type),
	}

	jdata, err := json.Marshal(e)
	if err != nil {
		return err
	}
	defer func() {
		rand.Read(jdata)
	}()

	return store.Set(BucketSecrets, secret.ID, jdata)
}

func (secret *Secret) Delete(store *Store) error {
	return store.Delete(BucketSecrets, secret.ID)
}

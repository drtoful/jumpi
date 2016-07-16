package jumpi

import (
	"crypto/rand"
	"encoding/json"
	"errors"

	"github.com/codahale/chacha20"
	"github.com/drtoful/jumpi/utils"
)

type TypeSecret int

const (
	Password TypeSecret = 0
	PKey
)

var (
	chachaRounds int = 20

	ErrUnknownSecretType = errors.New("unknown secret type")
)

type entry struct {
	Nonce  string `json:"nonce"`
	Rounds int    `json:"rounds"`
	Data   string `json:"data"`
	Type   int    `json:"type"`
}

type Secret struct {
	ID     string
	Type   TypeSecret
	Secret interface{}
}

func (secret *Secret) Load(store *Store) error {
	// load up encryption key
	data, err := store.Get(BucketSecretsKeys, secret.ID)
	if err != nil {
		return err
	}

	e := &entry{}
	if err := json.Unmarshal([]byte(data), e); err != nil {
		return err
	}

	nonce, err := utils.UnHexlify(e.Nonce)
	if err != nil {
		return err
	}

	key, err := utils.UnHexlify(e.Data)
	if err != nil {
		return err
	}

	skey, err := store.Password()
	if err != nil {
		return err
	}

	stream, err := chacha20.NewWithRounds(skey, nonce, uint8(e.Rounds))
	if err != nil {
		return err
	}
	stream.XORKeyStream(key, key)

	// decrypt actual secret
	data, err = store.Get(BucketSecrets, secret.ID)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(data), e); err != nil {
		return err
	}

	nonce, err = utils.UnHexlify(e.Nonce)
	if err != nil {
		return err
	}

	s, err := utils.UnHexlify(e.Data)
	if err != nil {
		return err
	}

	stream, err = chacha20.NewWithRounds(key, nonce, uint8(e.Rounds))
	if err != nil {
		return err
	}
	stream.XORKeyStream(s, s)

	// clear encryption key
	rand.Read(key)

	// try to convert recovered data, to the type
	// of the secret
	secret.Type = TypeSecret(e.Type)
	switch secret.Type {
	case Password:
		secret.Secret = string(s)
	}

	return nil
}

func (secret *Secret) Store(store *Store) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}

	nonceKey := make([]byte, 8)
	if _, err := rand.Read(nonceKey); err != nil {
		return err
	}

	nonceData := make([]byte, 8)
	if _, err := rand.Read(nonceData); err != nil {
		return err
	}

	// convert secret (whatever it is) to a byte stream, ready
	// to be encrypted
	var data []byte = nil
	switch s := secret.Secret.(type) {
	case string:
		data = []byte(s)
	default:
		return ErrUnknownSecretType
	}
	if data == nil {
		return ErrUnknownSecretType
	}

	// encrypt newly generated random key with the store password
	skey, err := store.Password()
	if err != nil {
		return err
	}

	stream, err := chacha20.NewWithRounds(skey, nonceKey, uint8(chachaRounds))
	if err != nil {
		return err
	}

	dkey := make([]byte, len(key))
	stream.XORKeyStream(dkey, key)
	e := &entry{
		Rounds: chachaRounds,
		Nonce:  utils.Hexlify(nonceKey),
		Data:   utils.Hexlify(dkey),
		Type:   0,
	}

	jdata, err := json.Marshal(e)
	if err != nil {
		return err
	}

	err = store.Set(BucketSecretsKeys, secret.ID, string(jdata))
	if err != nil {
		return err
	}

	// encrypt secret data with the newly generated random key
	stream, err = chacha20.NewWithRounds(key, nonceData, uint8(chachaRounds))
	if err != nil {
		return err
	}

	stream.XORKeyStream(data, data)
	e = &entry{
		Rounds: chachaRounds,
		Nonce:  utils.Hexlify(nonceData),
		Data:   utils.Hexlify(data),
		Type:   int(secret.Type),
	}

	jdata, err = json.Marshal(e)
	if err != nil {
		return err
	}

	err = store.Set(BucketSecrets, secret.ID, string(jdata))
	if err != nil {
		return err
	}

	// clear key by overwriting data (hopefully go memory managment
	// didn't create new memory space for that)
	rand.Read(key)

	return nil
}

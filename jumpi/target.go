package jumpi

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
)

type Target struct {
	Username string
	Hostname string
	Port     int
	Secret   *Secret

	store *Store
}

var (
	ErrNoSecret = errors.New("unable to locate secret for target")
)

func (target *Target) ID() string {
	return fmt.Sprintf("%s@%s:%d", target.Username, target.Hostname, target.Port)
}

func (target *Target) Store(store *Store) error {
	if target.Secret == nil {
		return ErrNoSecret
	}
	return store.Set(BucketTargets, target.ID(), target.Secret.ID)
}

func (target *Target) authPK() ([]ssh.Signer, error) {
	return nil, nil
}

func (target *Target) authPassword() (string, error) {
	if target.Secret == nil {
		return "", ErrNoSecret
	}

	if target.Secret.Type != Password {
		return "", nil
	}

	if target.Secret.Secret == nil {
		if err := target.Secret.Load(target.store); err != nil {
			return "", ErrNoSecret
		}
	}

	password, ok := target.Secret.Secret.(string)
	if !ok {
		return "", ErrNoSecret
	}

	return password, nil
}

func (target *Target) Connect(store *Store, newChannel ssh.NewChannel) error {
	clientConfig := &ssh.ClientConfig{
		User: target.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(target.authPK),
			ssh.PasswordCallback(target.authPassword),
		},
	}

	remote := fmt.Sprintf("%s:%d", target.Hostname, target.Port)
	client, err := ssh.Dial("tcp", remote, clientConfig)
	if err != nil {
		return err
	}

	channel, reqs, err := newChannel.Accept()
	if err != nil {
		return err
	}
	go ssh.DiscardRequests(reqs)

	channel2, _, err := client.OpenChannel("session", []byte{})
	if err != nil {
		return err
	}

	var closer sync.Once
	closeFunc := func() {
		channel.Close()
		channel2.Close()
		client.Close()
	}

	// copy stdin/stdout in cross pattern
	go func() {
		io.Copy(channel, channel2)
		closer.Do(closeFunc)
	}()

	go func() {
		io.Copy(channel2, channel)
		closer.Do(closeFunc)
	}()

	return nil
}

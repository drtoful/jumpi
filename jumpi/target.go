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
	Cast     *Cast

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
	return store.Set(BucketTargets, target.ID(), []byte(target.Secret.ID))
}

func (target *Target) LoadSecret(store *Store) error {
	if target.Secret != nil {
		return nil
	}

	secret, err := store.Get(BucketTargets, target.ID())
	if err != nil {
		return err
	}

	if len(secret) == 0 {
		return ErrNoSecret
	}

	target.Secret = &Secret{ID: string(secret)}
	if err := target.Secret.Load(store); err != nil {
		return err
	}

	return nil
}

func (target *Target) authPK() ([]ssh.Signer, error) {
	if target.Secret == nil {
		return nil, ErrNoSecret
	}

	if target.Secret.Type != PKey {
		return nil, nil
	}

	if target.Secret.Secret == nil {
		if err := target.Secret.Load(target.store); err != nil {
			return nil, err
		}
	}

	signer, err := ssh.NewSignerFromKey(target.Secret.Secret)
	if err != nil {
		return nil, err
	}

	return []ssh.Signer{signer}, nil
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
			return "", err
		}
	}

	password, ok := target.Secret.Secret.(string)
	if !ok {
		return "", ErrNoSecret
	}

	return password, nil
}

func (target *Target) proxy(reqs1, reqs2 <-chan *ssh.Request, channel1, channel2 ssh.Channel) {
	var closer sync.Once
	closerChan := make(chan bool, 1)

	closeFunc := func() {
		channel1.Close()
		channel2.Close()
	}
	defer closer.Do(closeFunc)

	go func() {
		target.Cast.Copy(channel1, channel2)
		closerChan <- true
	}()

	go func() {
		io.Copy(channel2, channel1)
		closerChan <- true
	}()

	for {
		select {
		case req := <-reqs1:
			if req == nil {
				return
			}
			b, err := channel2.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)
		case req := <-reqs2:
			if req == nil {
				return
			}
			b, err := channel1.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)
		case <-closerChan:
			return
		}
	}
}

func (target *Target) Connect(newChannel ssh.NewChannel, chans <-chan ssh.NewChannel) error {
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
	defer client.Close()

	sessChannel, sessReqs, err := newChannel.Accept()
	if err != nil {
		return err
	}
	defer sessChannel.Close()

	go func() {
		for newChannel := range chans {
			if newChannel == nil {
				return
			}

			channel2, reqs2, err := client.OpenChannel(newChannel.ChannelType(), newChannel.ExtraData())
			if err != nil {
				x, ok := err.(*ssh.OpenChannelError)
				if ok {
					newChannel.Reject(x.Reason, x.Message)
				} else {
					newChannel.Reject(ssh.Prohibited, "remote server denied channel request")
				}
				continue
			}

			channel, reqs, err := newChannel.Accept()
			if err != nil {
				channel2.Close()
				continue
			}
			go target.proxy(reqs, reqs2, channel, channel2)
		}
	}()

	channel2, reqs2, err := client.OpenChannel("session", []byte{})
	if err != nil {
		return err
	}

	target.proxy(sessReqs, reqs2, sessChannel, channel2)
	return nil
}

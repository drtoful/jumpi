package jumpi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

type Target struct {
	Username string
	Hostname string
	Port     int
	Secret   *Secret
	Cast     *Cast
	Session  string

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

func (target *Target) record(channel1, channel2 ssh.Channel, teardown chan bool, closer *sync.WaitGroup) {
	// this will only record data send back from server to client (the echo data
	// from the input and the actual terminal data)

	// data from server to client
	closer.Add(1)
	go func() {
		defer func() {
			log.Printf("ssh[%s]: stopping capture of data from server to client\n", target.Session)
			teardown <- true
			channel1.CloseWrite()
			closer.Done()
		}()
		log.Printf("ssh[%s]: starting capture of data from server to client\n", target.Session)
		target.Cast.Copy(channel1, channel2)
	}()

	// data from client to server
	closer.Add(1)
	go func() {
		defer func() {
			log.Printf("ssh[%s]: no more data from client, closing down server-client channel\n", target.Session)
			teardown <- true
			channel2.CloseWrite()
			closer.Done()
		}()
		io.Copy(channel2, channel1)
	}()
}

func (target *Target) proxy(reqs1, reqs2 <-chan *ssh.Request, channel1, channel2 ssh.Channel) {
	closer := &sync.WaitGroup{}
	teardown := make(chan bool)

	defer func() {
		log.Printf("ssh[%s]: tearing down channel\n", target.Session)
		go func() {
			// consume teardown channel
			for _ = range teardown {
			}
		}()

		// one of the channels has been teardowned, so close
		// all others
		channel1.Close()
		channel2.Close()

		// wait for all go-routines to close
		closer.Wait()
		close(teardown)
		log.Printf("ssh[%s]: teardown completed\n", target.Session)
	}()

	hasExec := false
	for {
		select {
		case req := <-reqs1:
			if req == nil {
				return
			}
			log.Printf("ssh[%s]: handling client request '%s'\n", target.Session, req.Type)
			b, err := channel2.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				log.Printf("ssh[%s]: unable to send client request '%s' to server: %s\n", target.Session, req.Type, err.Error())
				return
			}
			if req.WantReply {
				if err := req.Reply(b, nil); err != nil {
					log.Printf("ssh[%s]: error while waiting for response to '%s': %s\n", target.Session, req.Type, err.Error())
					return
				}
			}

			if req.Type == "exec" {
				hasExec = true

				// parse 'exec' payload which consists of the
				// channel id this occurs and the actual command
				var channel uint32
				buf := bytes.NewReader(req.Payload[:4])
				if err := binary.Read(buf, binary.BigEndian, &channel); err != nil {
					log.Printf("ssh[%s]: unable to parse channel number from 'exec' command\n", target.Session)
					return
				}
				cmd := string(req.Payload[4:])
				log.Printf("ssh[%s]: executing command on channel %d: %s\n", target.Session, channel, cmd)

				// handle 'scp' command
				if strings.HasPrefix(cmd, "scp") {
					log.Printf("ssh[%s]: detected secury copy command, starting interpreter\n", target.Session)
					target.handleSCP(cmd, channel1, channel2, teardown, closer)
					break
				}

				// for all other commands, we want to record the output
				target.record(channel1, channel2, teardown, closer)
			}

			// capture data, when client is requesting a 'shell'
			if req.Type == "shell" {
				target.record(channel1, channel2, teardown, closer)
			}

			break
		case req := <-reqs2:
			if req == nil {
				return
			}
			log.Printf("ssh[%s]: handling server request '%s'\n", target.Session, req.Type)
			b, err := channel1.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				log.Printf("ssh[%s]: unable to send server request '%s' to client: %s\n", target.Session, req.Type, err.Error())
				return
			}
			if req.WantReply {
				if err := req.Reply(b, nil); err != nil {
					log.Printf("ssh[%s]: error while waiting for response to '%s': %s\n", target.Session, req.Type, err.Error())
					return
				}
			}

			// handle 'exit-*' requests, concludes handling 'exec' request
			// from client, so we can complete teardown
			if hasExec && strings.HasPrefix(req.Type, "exit-") {
				// parse command exit status or signal
				switch req.Type {
				case "exit-status":
					var status uint32
					buf := bytes.NewReader(req.Payload[:4])
					if err := binary.Read(buf, binary.BigEndian, &status); err != nil {
						log.Printf("ssh[%s]: unable to parse exit status\n", target.Session)
						return
					}
					log.Printf("ssh[%s]: command exited with status %d\n", target.Session, status)
					break
				case "exit-signal":
					break
				}

				return
			}

			break
		case <-teardown:
			// if the client has issued an 'exec' request, we will have
			// to wait for the server to answer with a "exit-status" or
			// "exit-signal" request, before closing down channels
			if hasExec {
				log.Printf("ssh[%s]: client issued 'exec' request beforehand, will wait for server to respond\n", target.Session)
				continue
			}
			return
		}
	}
}

func (target *Target) Connect(sessChannel ssh.Channel, sessReqs <-chan *ssh.Request, chans <-chan ssh.NewChannel) error {
	clientConfig := &ssh.ClientConfig{
		User: target.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(target.authPK),
			ssh.PasswordCallback(target.authPassword),
		},

		// this is very trusting and just accepts connections
		// to everything. maybe add some sort of correct verification
		// in the future
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	remote := fmt.Sprintf("%s:%d", target.Hostname, target.Port)
	client, err := ssh.Dial("tcp", remote, clientConfig)
	if err != nil {
		return err
	}
	defer client.Close()

	var closer sync.WaitGroup
	go func() {
		for newChannel := range chans {
			if newChannel == nil {
				return
			}

			log.Printf("ssh[%s]: opening new channel %s\n", target.Session, newChannel.ChannelType())
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

			closer.Add(1)
			go func() {
				defer closer.Done()
				target.proxy(reqs, reqs2, channel, channel2)
			}()
		}
	}()

	log.Printf("ssh[%s]: opening 'session' channel on server\n", target.Session)
	channel2, reqs2, err := client.OpenChannel("session", []byte{})
	if err != nil {
		log.Printf("ssh[%s]: unable to open 'session' channel on server: %s\n", target.Session, err.Error())
		return err
	}

	closer.Add(1)
	go func() {
		defer closer.Done()
		target.proxy(sessReqs, reqs2, sessChannel, channel2)
	}()

	// wait for all sup-proxies to finish
	closer.Wait()
	return nil
}

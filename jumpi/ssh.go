package jumpi

import (
	"crypto/sha256"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/drtoful/jumpi/utils"
	"golang.org/x/crypto/ssh"
)

type server struct {
	store  *Store
	config *ssh.ServerConfig
}

var (
	ErrNoHostKey = errors.New("no host key found")
)

// parse a target declaration in the form user@host[:port]
func (server *server) parseTarget(id string) *Target {
	var user string
	var port int = 22
	var host string

	splits := strings.Split(id, "@")
	if len(splits) == 2 {
		user = splits[0]
		id = splits[1]
	} else {
		return nil
	}

	splits = strings.Split(id, ":")
	if len(splits) == 2 {
		host = splits[0]
		if i, err := strconv.ParseInt(splits[1], 10, 32); err == nil {
			return nil
		} else {
			port = int(i)
		}
	} else if len(splits) == 1 {
		host = id
	} else {
		return nil
	}

	target := &Target{
		Username: user,
		Hostname: host,
		Port:     port,
	}
	if err := target.LoadSecret(server.store); err != nil {
		return nil
	}
	return target
}

func (server *server) handle(conn net.Conn) {
	log.Printf("ssh: new connection from %s\n", conn.RemoteAddr().String())
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, server.config)
	if err != nil {
		log.Printf("ssh: error for '%s': %s\n", conn.RemoteAddr().String(), err.Error())
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	newChannel := <-chans
	if newChannel == nil {
		log.Printf("ssh: error for '%s': no channel found\n", conn.RemoteAddr().String())
		return
	}

	var target *Target
	if newChannel.ChannelType() == "session" {
		target = server.parseTarget(sshConn.User())
	}

	if target == nil {
		return
	}

	if err := target.Connect(newChannel, chans); err != nil {
		log.Printf("ssh: error for '%s': %s\n", conn.RemoteAddr().String(), err.Error())
	}
}

func (server *server) serve() error {
	conn, err := net.Listen("tcp", ":2022")
	if err != nil {
		return err
	}

	go func() {
		for {
			client, err := conn.Accept()
			if err != nil {
				log.Printf("ssh connect error: %s\n", err.Error())
				continue
			}

			go server.handle(client)
		}
		conn.Close()
	}()

	return nil
}

func (server *server) auth(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	k := key.Marshal()
	t := key.Type()
	perm := &ssh.Permissions{
		Extensions: map[string]string{
			"pubKey":     string(k),
			"pubKeyType": t,
		},
	}

	digest := sha256.New()
	digest.Write(k)
	id := utils.Hexlify(digest.Sum(nil))
	user := &User{KeyFingerprint: id}
	if err := user.Load(server.store); err != nil {
		return nil, err
	}
	perm.Extensions["user"] = user.Name

	return perm, nil
}

func StartSSHServer(store *Store, hostkey string) error {
	// try to load hostkey from file
	data, err := ioutil.ReadFile(hostkey)
	if err != nil {
		return err
	}

	pkey, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return err
	}

	server := &server{store: store}
	server.config = &ssh.ServerConfig{
		PublicKeyCallback: server.auth,
	}
	server.config.AddHostKey(pkey)

	return server.serve()
}

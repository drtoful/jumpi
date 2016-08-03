package jumpi

import (
	"crypto/rand"
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
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, server.config)
	if err != nil {
		log.Printf("ssh[main]: unable to create SSH connection for '%s': %s\n", conn.RemoteAddr().String(), err.Error())
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	perm := sshConn.Permissions
	session := perm.Extensions["session"]
	log.Printf("ssh[%s]: new connection from %s\n", session, conn.RemoteAddr().String())

	newChannel := <-chans
	if newChannel == nil {
		log.Printf("ssh[%s]: error: no channel found\n", session)
		return
	}

	var target *Target
	if newChannel.ChannelType() == "session" {
		target = server.parseTarget(sshConn.User())
	}

	if target == nil {
		return
	}

	log.Printf("ssh[%s]: connecting to %s\n", session, target.ID())
	if err := target.Connect(newChannel, chans); err != nil {
		log.Printf("ssh[%s]: error: %s\n", session, err.Error())
	}
	log.Printf("ssh[%s]: session ended\n", session)
}

func (server *server) serve() error {
	log.Println("starting SSH server on port 2022")
	conn, err := net.Listen("tcp", ":2022")
	if err != nil {
		return err
	}

	go func() {
		for {
			client, err := conn.Accept()
			if err != nil {
				log.Printf("ssh[main]: connect error: %s\n", err.Error())
				continue
			}

			go server.handle(client)
		}
		conn.Close()
	}()

	return nil
}

func generateSessionID() (string, error) {
	sess := make([]byte, 16)
	if _, err := rand.Read(sess); err != nil {
		return "", err
	}

	return utils.Hexlify(sess), nil
}

func (server *server) auth(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	session, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	k := key.Marshal()
	t := key.Type()
	perm := &ssh.Permissions{
		Extensions: map[string]string{
			"pubKey":     string(k),
			"pubKeyType": t,
			"session":    session,
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
	log.Printf("ssh[%s]: user '%s' successfully logged on\n", session, user.Name)

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

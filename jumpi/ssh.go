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
func (server *server) parseTarget(session, id string) *Target {
	var user string
	var port int = 22
	var host string

	splits := strings.Split(id, "@")
	if len(splits) == 2 {
		user = splits[0]
		id = splits[1]
	} else {
		log.Printf("ssh[%s]: incorrect jump target format ('%s')\n", session, id)
		return nil
	}

	splits = strings.Split(id, ":")
	if len(splits) == 2 {
		host = splits[0]
		if i, err := strconv.ParseInt(splits[1], 10, 32); err != nil {
			log.Printf("ssh[%s]: unable to parse port number ('%s'): %s\n", session, id, err.Error())
			return nil
		} else {
			port = int(i)
		}
	} else if len(splits) == 1 {
		host = id
	} else {
		log.Printf("ssh[%s]: no hostname provided ('%s')\n", session, id)
		return nil
	}

	target := &Target{
		Username: user,
		Hostname: host,
		Port:     port,
	}
	if err := target.LoadSecret(server.store); err != nil {
		log.Printf("ssh[%s]: unable to load secret for target '%s': %s\n", session, id, err.Error())
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
	user := perm.Extensions["user"]
	log.Printf("ssh[%s]: new connection from %s\n", session, conn.RemoteAddr().String())
	defer log.Printf("ssh[%s]: session ended \n", session)

	newChannel := <-chans
	if newChannel == nil {
		log.Printf("ssh[%s]: error: no channel found\n", session)
		return
	}

	ok, role := CheckRole(user, sshConn.User())
	if !ok {
		log.Printf("ssh[%s]: permission denied to access '%s'\n", session, sshConn.User())
		return
	}
	log.Printf("ssh[%s]: user allowed to access target by role '%s'\n", session, role)

	var target *Target
	if newChannel.ChannelType() == "session" {
		target = server.parseTarget(session, sshConn.User())
	}

	if target == nil {
		log.Printf("ssh[%s]: unable to parse target '%s'\n", session, sshConn.User())
		return
	}

	target.Cast = &Cast{
		Session: session,
		User:    user,
		Target:  target.ID(),
	}
	if err := target.Cast.Start(server.store); err != nil {
		log.Printf("ssh[%s]: error: %s\n", session, err.Error())
		return
	}
	log.Printf("ssh[%s]: starting recording of session\n", session)

	log.Printf("ssh[%s]: connecting to %s\n", session, target.ID())
	if err := target.Connect(newChannel, chans); err != nil {
		log.Printf("ssh[%s]: error: %s\n", session, err.Error())
	}

	// stop recording and store
	log.Printf("ssh[%s]: stopped recording of session\n", session)
	target.Cast.Stop()
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

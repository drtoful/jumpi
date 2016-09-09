package jumpi

// implements a asciicast to be later used with asciinema-player
//    (https://github.com/asciinema/asciinema-player)

import (
	"bufio"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/codahale/chacha20"
	"github.com/drtoful/jumpi/utils"
)

var (
	ErrNoSession = errors.New("unable to start cast recording: no session set")
)

type castEntry struct {
	Data  string
	Delay float64
}

type secFile struct {
	secret []byte
	nonce  []byte
	rounds uint8
	fd     *os.File
	stream cipher.Stream
}

type Cast struct {
	Session  string          `json:"-"`
	Duration float64         `json:"duration"`
	Records  [][]interface{} `json:"stdout"`
	Width    int             `json:"width"`
	Height   int             `json:"height"`
	Version  int             `json:"version"`

	recorder chan *castEntry
	file     *secFile
}

func (f *secFile) Write(data []byte) (int, error) {
	f.stream.XORKeyStream(data, data)
	return f.fd.Write(data)
}

func (f *secFile) Read(buf []byte) (int, error) {
	n, err := f.fd.Read(buf)
	if err != nil {
		return n, err
	}

	f.stream.XORKeyStream(buf[0:n], buf[0:n])
	return n, err
}

func (f *secFile) Reset() error {
	stream, err := chacha20.NewWithRounds(f.secret, f.nonce, f.rounds)
	if err != nil {
		return err
	}

	f.stream = stream
	if _, err := f.fd.Seek(0, 0); err != nil {
		return err
	}

	return nil
}

func (f *secFile) Close() {
	f.fd.Close()
	os.Remove(f.fd.Name())
}

func newSecFile() (*secFile, error) {
	tmpfile, err := ioutil.TempFile("", "jumpi")
	if err != nil {
		return nil, err
	}
	file := &secFile{fd: tmpfile, rounds: 20}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	file.secret = key
	file.nonce = nonce

	if err := file.Reset(); err != nil {
		return nil, err
	}

	return file, nil
}

// similar to io.copyBuffer method, but instead of directly writing to
// the destination, we will keep a copy and safe it to be replayed
// later
func (cast *Cast) Copy(dest io.Writer, src io.Reader) (int64, error) {
	var written int64
	var err error

	buf := make([]byte, 32<<10) // 32KB
	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			//save to cast
			entry := &castEntry{
				Data: utils.Hexlify(buf[0:nr]),
			}
			go func() {
				cast.recorder <- entry
			}()

			nw, ew := dest.Write(buf[0:nr]) // only write what was read
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if err == io.EOF {
			err = nil // EOF is no real error
			break
		}
		if err != nil {
			break
		}
	}

	return written, err
}

func (cast *Cast) Start() error {
	if len(cast.Session) == 0 {
		return ErrNoSession
	}

	// set defaults
	cast.Version = 1
	cast.Width = 80
	cast.Height = 24

	fd, err := newSecFile()
	if err != nil {
		return err
	}
	cast.file = fd
	cast.recorder = make(chan *castEntry)
	log.Printf("ssh[%s]: storing recording into %s\n", cast.Session, cast.file.fd.Name())

	go func() {
		for {
			start := time.Now().UTC()
			e := <-cast.recorder
			end := time.Now().UTC()

			if e == nil {
				break
			}

			e.Delay = float64(end.Sub(start)) / float64(time.Second)
			jdata, err := json.Marshal(e)
			if err == nil {
				cast.file.Write(jdata)
				cast.file.Write([]byte{'\n'})
			}

			cast.Duration += e.Delay
		}
	}()

	return nil
}

func (cast *Cast) Stop() {
	if cast.recorder == nil {
		return
	}
	cast.recorder <- nil
}

func (cast *Cast) Store(store *Store) error {
	// move to top of file and reset chacha20 stream
	cast.file.Reset()
	cast.Records = make([][]interface{}, 0)
	defer cast.file.Close()

	// read json objects line by line and add them to the
	// object and then store it in the database
	reader := bufio.NewReader(cast.file)
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		var entry castEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return err
		}

		data, err := utils.UnHexlify(entry.Data)
		if err != nil {
			return err
		}
		cast.Records = append(cast.Records, []interface{}{entry.Delay, string(data)})
	}

	// store into database
	jdata, err := json.Marshal(cast)
	if err != nil {
		return err
	}
	defer func() {
		rand.Read(jdata)
	}()
	return store.Set(BucketCasts, cast.Session, jdata)
}

package jumpi

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

var (
	sourceMode = regexp.MustCompile("\\-[^f\\-\\s]*f[^f\\-\\s]*")
	sinkMode   = regexp.MustCompile("\\-[^t\\-\\s]*t[^t\\-\\s]*")

	ErrWrongHeader    = errors.New("unable to parse SCP header, may be corruped")
	ErrUnknownCommand = errors.New("unknown SCP header command")
)

type scp struct {
	session string
}

type file_info struct {
	name string
	mode string
	size int
}

func (s *scp) Copy(dest io.Writer, src io.Reader) (int64, error) {
	var written int64
	var err error
	var i, to_copy int
	var read_header bool
	var current_file file_info

	read_header = true
	buf := make([]byte, 32<<10) // 32KB
	head_buf := bytes.NewBuffer(make([]byte, 4<<10))
	digest := sha512.New()
	for {
		// read the next bunch of data
		nr, err := src.Read(buf)
		if nr > 0 {
			// copy the data through the wire
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

			// interpret the read data
			i = 0
			for {
				if i >= nr {
					break
				}

				// we're currently reading the header
				if read_header {
					// search for '\n' in buf
					var k int
					var found = false
					for k = i; k < nr; k += 1 {
						if k >= nr {
							break
						}
						if buf[k] == '\n' {
							found = true
							break
						}
					}

					// write to buffer
					head_buf.Write(buf[i:k])

					// advance the pointer in the buffer
					i = k + 1
					if !found {
						// we did not yet find the end of the header,
						// so we continue searching
						continue
					}

					// try to determine size to copy
					pieces := strings.Split(head_buf.String(), " ")
					if len(pieces) > 0 {
						// determine command
						cmd := strings.TrimLeft(pieces[0], "\x00\x01\x02")
						switch cmd[0] {
						case 'D': // creating directories
							break
						case 'C': // creating files
							if len(pieces) < 3 {
								return 0, ErrWrongHeader
							}

							v, err := strconv.ParseInt(pieces[1], 10, 64)
							if err != nil {
								return 0, err
							}
							to_copy = int(v)
							read_header = false
							digest.Reset()

							current_file = file_info{
								size: to_copy,
								mode: string(cmd[1:]),
								name: strings.Join(pieces[2:], " "),
							}
							break
						case 'E': // pop from directory stack
							break
						case 'T': // command not yet implemented
							break
						default:
							return 0, ErrUnknownCommand
						}

					} else {
						return 0, ErrWrongHeader
					}

					head_buf.Reset()
				} else {
					min := int(math.Min(float64(nr-i), float64(to_copy)))
					digest.Write(buf[i:min])

					// reduce the target number to write
					to_copy -= min
					i += min
					if to_copy == 0 {
						read_header = true
						i += 1 // read over last byte?
						log.Printf("scp[%s]: transfered file '%s' (%d bytes) with mode %s: sha512=%s\n", s.session, current_file.name, current_file.size, current_file.mode, hex.EncodeToString(digest.Sum(nil)))
					}
				}
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

func (target *Target) handleSCP(cmd string, channel1, channel2 ssh.Channel, teardown chan bool, closer *sync.WaitGroup) {
	isSource := false
	isSink := false

	arguments := strings.Split(cmd, " ")
	if len(arguments) < 2 {
		log.Printf("scp[%s]: unable to get arguments of scp\n", target.Session)
		return
	}

	if sourceMode.MatchString(arguments[1]) {
		isSource = true
		log.Printf("scp[%s]: detected source mode copy: client <- server\n", target.Session)
	}

	if sinkMode.MatchString(arguments[1]) {
		isSink = true
		log.Printf("scp[%s]: detected sink mode copy: client -> server\n", target.Session)
	}

	if !isSource && !isSink {
		log.Printf("scp[%s]: unable to detect mode\n", target.Session)
		return
	}

	if isSource && isSink {
		log.Printf("scp[%s]: detected both sink and source mode. something went wrong\n", target.Session)
		return
	}

	scp := &scp{
		session: target.Session,
	}

	closer.Add(1)
	go func() {
		defer func() {
			teardown <- true
			channel1.CloseWrite()
			closer.Done()
		}()
		if isSource {
			_, err := scp.Copy(channel1, channel2)
			if err != nil {
				log.Printf("scp[%s]: unable to parse scp: %s\n", target.Session, err.Error())
			}
		} else {
			io.Copy(channel1, channel2)
		}
	}()

	closer.Add(1)
	go func() {
		defer func() {
			teardown <- true
			channel2.CloseWrite()
			closer.Done()
		}()
		if isSink {
			_, err := scp.Copy(channel2, channel1)
			if err != nil {
				log.Printf("scp[%s]: unable to parse scp: %s\n", target.Session, err.Error())
			}
		} else {
			io.Copy(channel2, channel1)
		}
	}()
}

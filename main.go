package main

//go:generate go-bindata -pkg "jumpi" -o jumpi/bindata.go static/...

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/drtoful/jumpi/jumpi"
	"github.com/drtoful/jumpi/utils/mlock"
)

var (
	mlockOpt   = flag.Bool("mlock", false, "enable MLock")
	dbOpt      = flag.String("db", "jumpi.db", "path to jumpi database file")
	hostKeyOpt = flag.String("hostkey", "id_rsa", "path to host key to use for SSH server")
)

func main() {
	flag.Parse()

	// since we deal with passwords etc. we will lock the memory in place
	// so it will not get swapped out, and can thus be read afterwards.
	if *mlockOpt {
		if mlock.Supported() {
			if err := mlock.LockMemory(); err != nil {
				log.Fatalf("mlock init error: %s\n", err.Error())
			}
		} else {
			log.Println("MLock is unavailable to this Operating system, will continue without")
		}
	}

	// check if database does already exist. if not, then this is a
	// first time run
	var ftr bool = false
	if _, err := os.Stat(*dbOpt); os.IsNotExist(err) {
		ftr = true
	}

	// create a new database
	store, err := jumpi.NewStore(*dbOpt)
	if err != nil {
		log.Fatalf("db init error: %s\n", err.Error())
	}
	defer store.Close()

	if ftr {
		store.FTR()
	}

	// start all services
	jumpi.StartAPIServer("/", store)
	if err := jumpi.StartSSHServer(store, *hostKeyOpt); err != nil {
		log.Fatalf("unable to start SSH server: %s\n", err.Error())
	}

	// all listeners are started in the background as
	// gofunc's so we wait here for an interupt signal
	// to stop the service gracefully
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-sig:
			log.Fatalf("main: Signal (%d) received, stopping\n", s)
		}
	}
}

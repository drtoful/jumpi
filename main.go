package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/drtoful/jumpi/jumpi"
	"github.com/drtoful/jumpi/utils/mlock"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	mlockOpt = flag.Bool("mlock", false, "enable MLock")
	dbOpt    = flag.String("db", "jumpi.db", "path to jumpi database file")
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

	// create a new database
	store, err := jumpi.NewStore(*dbOpt)
	if err != nil {
		log.Fatalf("db init error: %s\n", err.Error())
	}
	defer store.Close()

	// unlock store, prompt for password
	fmt.Printf("unlock password: ")
	pwd, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	if err := store.Unlock(string(pwd)); err != nil {
		log.Fatalf("unable to unlock store: %s\n", err.Error())
	}

	// start all services
	jumpi.StartAPIServer("/", store)

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

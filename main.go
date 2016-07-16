package main

import (
	"flag"
	"log"

	"github.com/drtoful/jumpi/jumpi"
	"github.com/drtoful/jumpi/utils/mlock"
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
}

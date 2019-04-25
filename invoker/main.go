package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	// Assign ID from command line arg

	invokerIDPtr := flag.Int("id", -1, "unique id for the invoker")
	flag.Parse()

	if *invokerIDPtr < 0 {
		fmt.Fprintln(os.Stderr, "Invoker must have a unique id assigned greater than 0")
		os.Exit(1)
	}
	invokerID := *invokerIDPtr
	fmt.Printf("Invoker with id: %d starting.", invokerID)

	// Create/ensure topic for invoker i

	// Send ping to controller with name :)

	// Pull messages from invoker queue

	// Put messages back on completed i queue

}

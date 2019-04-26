package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ostenbom/refunction/invoker/messages"
)

const healthTopic = "health"

func main() {

	// Assign ID from command line arg

	invokerIDPtr := flag.Int("id", -1, "unique id for the invoker")
	flag.Parse()

	if *invokerIDPtr < 0 {
		fmt.Fprintln(os.Stderr, "Invoker must have a unique id assigned greater than 0")
		os.Exit(1)
	}
	invokerID := fmt.Sprintf("invoker%d", *invokerIDPtr)
	fmt.Printf("Invoker with id: %s starting\n", invokerID)

	// Create/ensure topic for invoker i
	provider, err := messages.NewMessageProvider("172.17.0.1:9093")
	if err != nil {
		errExit(fmt.Sprintf("could not create message provider: %s", err))
	}
	defer provider.Close()

	err = provider.EnsureTopic(invokerID)
	if err != nil {
		errExit(fmt.Sprintf("could not ensure invokers topic: %s", err))
	}

	fmt.Println("successfully ensured the topic")

	// Send ping to controller with name :)

	// Pull messages from invoker queue

	// Put messages back on completed i queue

}

func errExit(err string) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

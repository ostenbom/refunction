package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ostenbom/refunction/invoker/messages"
)

const healthTopic = "health"
const twoGigMem = "2147483648 B"

type Ping struct {
	Name PingName `json:"name"`
}

type PingName struct {
	Instance   int    `json:"instance"`
	UniqueName string `json:"uniqueName"`
	UserMemory string `json:"userMemory"`
}

func main() {

	// Assign ID from command line arg

	invokerIDPtr := flag.Int("id", -1, "unique id for the invoker")
	flag.Parse()

	if *invokerIDPtr < 0 {
		fmt.Fprintln(os.Stderr, "Invoker must have a unique id assigned greater than 0")
		os.Exit(1)
	}
	invokerNumber := *invokerIDPtr
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

	heathStop := startHealthPings(invokerNumber, provider)
	defer func() {
		heathStop <- true
	}()

	time.Sleep(time.Second * 5)

	// Pull messages from invoker queue

	// Put messages back on completed i queue

}

func startHealthPings(invokerNumber int, provider messages.MessageProvider) chan bool {
	stop := make(chan bool)
	go func() {
		for {
			select {
			default:
				go func() {
					err := sendPing(invokerNumber, provider)
					if err != nil {
						fmt.Fprintf(os.Stderr, "health ping failure: %s", err)
					}
				}()
				time.Sleep(time.Second)
			case <-stop:
				return
			}
		}
	}()

	return stop
}

func sendPing(invokerNumber int, provider messages.MessageProvider) error {
	msg := Ping{
		Name: PingName{
			Instance:   invokerNumber,
			UniqueName: fmt.Sprintf("%d", invokerNumber),
			UserMemory: twoGigMem,
		},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("could not marshal ping message: %s", err)
	}

	err = provider.WriteMessage("health", msgBytes)
	if err != nil {
		return fmt.Errorf("could not send ping message: %s", err)
	}

	return nil
}

func errExit(err string) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

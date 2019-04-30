package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ostenbom/refunction/invoker/messages"
	"github.com/ostenbom/refunction/invoker/storage"
)

const healthTopic = "health"
const twoGigMem = "2147483648 B"

const defaultKafkaAddress = "172.17.0.1:9093"
const defaultCouchDBAddress = "http://admin:specialsecretpassword@127.0.0.1:5984"
const defaultActivationDBName = "whisk_local_activations"
const defaultFunctionDBName = "whisk_local_whisks"

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
	// invokerNumber := *invokerIDPtr
	invokerID := fmt.Sprintf("invoker%d", *invokerIDPtr)
	fmt.Printf("Invoker with id: %s starting\n", invokerID)

	// Create/ensure topic for invoker i
	messageProvider, err := messages.NewMessageProvider(defaultKafkaAddress)
	if err != nil {
		errExit(fmt.Errorf("could not create message messageProvider: %s", err))
	}
	defer messageProvider.Close()

	err = messageProvider.EnsureTopic(invokerID)
	if err != nil {
		errExit(fmt.Errorf("could not ensure invokers topic: %s", err))
	}

	functionStorage, err := storage.NewFunctionStorage(defaultCouchDBAddress, defaultFunctionDBName, defaultActivationDBName)
	if err != nil {
		errExit(fmt.Errorf("could not establish couch connection: %s", err))
	}

	// healthStop := startHealthPings(invokerNumber, provider)
	// defer func() {
	// 	healthStop <- true
	// }()

	// Start fixed group of workers.

	for {
		// Pull messages from invoker queue
		message, err := messageProvider.ReadMessage(invokerID)
		if err != nil {
			errExit(fmt.Errorf("could not pull from invoker queue: %s", err))
		}

		var activation messages.ActivationMessage
		err = json.Unmarshal(message, &activation)
		if err != nil {
			errExit(fmt.Errorf("could not parse activation message: %s", err))
		}
		fmt.Printf("Action name: %s, ActivationID: %s\n", activation.Action.Name, activation.ActivationID)

		// Fetch required function
		function, err := functionStorage.GetFunction(activation.Action.Path, activation.Action.Name)
		if err != nil {
			errExit(fmt.Errorf("could not get activation function: %s", err))
		}

		fmt.Printf("Function Code: %s\n", function.Executable.Code)

		// Schedule function

		// Send ack
	}

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

func errExit(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

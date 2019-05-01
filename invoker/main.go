package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ostenbom/refunction/invoker/messages"
	"github.com/ostenbom/refunction/invoker/storage"
	"github.com/ostenbom/refunction/invoker/workerpool"
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

func startInvoker() int {

	// Assign ID from command line arg

	invokerIDPtr := flag.Int("id", -1, "unique id for the invoker")
	flag.Parse()

	if *invokerIDPtr < 0 {
		printError(fmt.Errorf("Invoker must have a unique id assigned greater than 0"))
		return 1
	}
	invokerID := fmt.Sprintf("invoker%d", *invokerIDPtr)
	fmt.Printf("Invoker with id: %s starting\n", invokerID)

	// Create/ensure topic for invoker i
	messageProvider, err := messages.NewMessageProvider(defaultKafkaAddress)
	if err != nil {
		printError(fmt.Errorf("could not create message messageProvider: %s", err))
		return 1
	}
	defer messageProvider.Close()

	err = messageProvider.EnsureTopic(invokerID)
	if err != nil {
		printError(fmt.Errorf("could not ensure invokers topic: %s", err))
		return 1
	}

	functionStorage, err := storage.NewFunctionStorage(defaultCouchDBAddress, defaultFunctionDBName, defaultActivationDBName)
	if err != nil {
		printError(fmt.Errorf("could not establish couch connection: %s", err))
		return 1
	}

	// invokerNumber := *invokerIDPtr
	// healthStop := startHealthPings(invokerNumber, messageProvider)
	// defer func() {
	// 	healthStop <- true
	// }()

	// Start fixed group of workers.
	workers, err := workerpool.NewWorkerPool(1)
	if err != nil {
		printError(err)
		return 1
	}
	defer workers.Close()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	messageChan := make(chan []byte)
	errorChan := make(chan error)

	go func() {
		for {
			message, err := messageProvider.ReadMessage(invokerID)
			if err != nil {
				errorChan <- fmt.Errorf("could not pull from invoker queue: %s", err)
				return
			}
			messageChan <- message
		}
	}()

	// Graceful stopping in infinite loop
	for {
		select {
		case <-stopChan:
			fmt.Println("shutting down")
			return 0
		case message := <-messageChan:
			err = consumeMessage(message, functionStorage, workers)
			if err != nil {
				printError(fmt.Errorf("could not consume message %s: %s", string(message), err))
				return 1
			}
		case err := <-errorChan:
			printError(err)
			return 1
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}

	// return 0
}

func consumeMessage(message []byte, functionStorage storage.FunctionStorage, workers *workerpool.WorkerPool) error {
	var activation messages.ActivationMessage
	err := json.Unmarshal(message, &activation)
	if err != nil {
		return fmt.Errorf("could not parse activation message: %s", err)
	}
	fmt.Printf("Action name: %s, ActivationID: %s\n", activation.Action.Name, activation.ActivationID)

	// Fetch required function
	function, err := functionStorage.GetFunction(activation.Action.Path, activation.Action.Name)
	if err != nil {
		return fmt.Errorf("could not get activation function: %s", err)
	}

	fmt.Printf("Function Code: %s\n", function.Executable.Code)

	// Schedule function
	result, err := workers.Run(function, "")
	if err != nil {
		return fmt.Errorf("could not run function %s: %s", function.Name, err)
	}

	fmt.Printf("ran function! Result %s\n", result)

	// Send ack

	return nil
}

func main() {
	exitCode := startInvoker()
	os.Exit(exitCode)
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

func printError(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
}

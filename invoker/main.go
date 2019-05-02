package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ostenbom/refunction/invoker/messages"
	"github.com/ostenbom/refunction/invoker/storage"
	"github.com/ostenbom/refunction/invoker/workerpool"
	log "github.com/sirupsen/logrus"
)

const defaultCouchDBAddress = "http://admin:specialsecretpassword@127.0.0.1:5984"
const defaultActivationDBName = "whisk_local_activations"
const defaultFunctionDBName = "whisk_local_whisks"

func startInvoker() int {

	// Assign ID from command line arg

	invokerIDPtr := flag.Int("id", -1, "unique id for the invoker")
	flag.Parse()

	if *invokerIDPtr < 0 {
		printError(fmt.Errorf("Invoker must have a unique id assigned greater than 0"))
		return 1
	}
	invokerID := fmt.Sprintf("invoker%d", *invokerIDPtr)
	log.Info(fmt.Sprintf("Invoker with id: %s starting", invokerID))

	invokerNumber := *invokerIDPtr
	messenger, err := messages.NewMessenger(invokerNumber)
	if err != nil {
		printError(err)
		return 1
	}
	defer messenger.Close()

	log.Debug("Messenger initialized")

	functionStorage, err := storage.NewFunctionStorage(defaultCouchDBAddress, defaultFunctionDBName, defaultActivationDBName)
	if err != nil {
		printError(fmt.Errorf("could not establish couch connection: %s", err))
		return 1
	}

	log.Debug("Function storage connected")

	healthStop := messenger.StartHealthPings(invokerNumber)
	defer func() {
		healthStop <- true
	}()

	// Start fixed group of workers.
	workers, err := workerpool.NewWorkerPool(1)
	if err != nil {
		printError(err)
		return 1
	}
	defer workers.Close()

	log.Info("Invoker initialized")

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	messageChan := make(chan *messages.ActivationMessage)
	errorChan := make(chan error)

	go func() {
		for {
			message, err := messenger.GetActivation()
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
			log.Info("Shutting Down")
			return 0
		case message := <-messageChan:
			// TODO: non-blocking
			err = consumeMessage(message, functionStorage, workers)
			if err != nil {
				printError(fmt.Errorf("could not consume message %s: %s", message.Action.Name, err))
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

func consumeMessage(activation *messages.ActivationMessage, functionStorage storage.FunctionStorage, workers *workerpool.WorkerPool) error {
	// Fetch required function
	function, err := functionStorage.GetFunction(activation.Action.Path, activation.Action.Name)
	if err != nil {
		return fmt.Errorf("could not get activation function: %s", err)
	}

	log.WithFields(log.Fields{
		"code": function.Executable.Code,
	}).Debug("fetched function")

	// Schedule function
	result, err := workers.Run(function, "{}")
	if err != nil {
		return fmt.Errorf("could not run function %s: %s", function.Name, err)
	}

	log.WithFields(log.Fields{
		"result": result,
	}).Debug("function run complete")

	// Send ack

	return nil
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	exitCode := startInvoker()
	os.Exit(exitCode)
}

func printError(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
}

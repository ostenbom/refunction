package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
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

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "172.17.0.1:9093",
		"group.id":          "oliversSpecialGroup",
	})

	if err != nil {
		panic(err)
	}

	c.SubscribeTopics([]string{"invoker0", "completed0"}, nil)

	for {
		msg, err := c.ReadMessage(-1)
		if err == nil {
			fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
		} else {
			// The client will automatically try to recover from all errors.
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

	c.Close()
}

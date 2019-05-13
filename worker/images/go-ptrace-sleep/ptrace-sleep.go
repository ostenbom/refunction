package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func main() {
	// Allow graceful stopping
	stopSigs := make(chan os.Signal, 1)
	signal.Notify(stopSigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stopSigs
		os.Exit(0)
	}()

	startedMessage := Message{
		Type: "started",
		Data: "",
	}
	messageJSON, err := json.Marshal(startedMessage)
	if err != nil {
		panic("could not marshal message")
	}
	fmt.Println(string(messageJSON))

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var message Message
		err := json.Unmarshal(scanner.Bytes(), &message)
		if err != nil {
			continue
		}
		if message.Type == "go" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

	f, openErr := os.OpenFile("/tmp/count.txt", os.O_CREATE|os.O_TRUNC|os.O_APPEND|os.O_WRONLY, 0600)
	if openErr != nil {
		panic(openErr)
	}
	defer f.Close()

	count := 0
	for true {
		fmt.Println("sleeping for 2")
		time.Sleep(time.Second * 2)

		_, err := f.WriteString(strconv.Itoa(count))
		if err != nil {
			panic(err)
		}

		count++
	}
}

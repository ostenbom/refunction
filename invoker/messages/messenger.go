package messages

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ostenbom/refunction/invoker/types"
	log "github.com/sirupsen/logrus"
)

const defaultKafkaAddress = "172.17.0.1:9093"
const healthTopic = "health"
const twoGigMem = "2147483648 B"

type Ping struct {
	Name types.Name `json:"name"`
}

type Messenger struct {
	provider      MessageProvider
	invokerNumber int
	invokerTopic  string
	name          types.Name
}

func NewMessenger(invokerNumber int) (*Messenger, error) {
	invokerTopic := fmt.Sprintf("invoker%d", invokerNumber)
	provider, err := NewMessageProvider(defaultKafkaAddress)
	if err != nil {
		return nil, err
	}
	err = provider.EnsureTopic(invokerTopic)
	if err != nil {
		return nil, fmt.Errorf("could not ensure topic for invoker %d: %s", invokerNumber, err)
	}
	return &Messenger{
		provider:      provider,
		invokerNumber: invokerNumber,
		invokerTopic:  invokerTopic,
		name: types.Name{
			Instance:   invokerNumber,
			UniqueName: fmt.Sprintf("%d", invokerNumber),
			UserMemory: twoGigMem,
		},
	}, nil
}

func (m *Messenger) GetActivation() (*types.ActivationMessage, error) {
	rawMessage, err := m.provider.ReadMessage(m.invokerTopic)
	if err != nil {
		return nil, fmt.Errorf("could not pull from invoker queue: %s", err)
	}

	var activation types.ActivationMessage
	err = json.Unmarshal(rawMessage, &activation)
	if err != nil {
		return nil, fmt.Errorf("could not parse activation message: %s", err)
	}

	log.WithFields(log.Fields{
		"name": activation.Action.Name,
		"ID":   activation.ActivationID,
	}).Debug("received activation message")

	return &activation, nil
}

func (m *Messenger) SendCompletion(activation *types.ActivationMessage, function *types.FunctionDoc, result string) error {
	controllerTopic := fmt.Sprintf("completed%s", activation.Controller.AsString)

	completion := types.CompletionMessage{
		ActivationID:  activation.ActivationID,
		Invoker:       m.name,
		SystemError:   false,
		TransactionID: activation.TransactionID,
	}

	rawCompletion, err := json.Marshal(completion)
	if err != nil {
		return fmt.Errorf("could not marshal completion message: %s", err)
	}
	return m.provider.WriteMessage(controllerTopic, rawCompletion)
}

func (m *Messenger) StartHealthPings(invokerNumber int) chan bool {
	stop := make(chan bool)
	go func() {
		for {
			select {
			default:
				go func() {
					err := m.sendPing(invokerNumber)
					if err != nil {
						log.Error(fmt.Sprintf("health ping failure: %s", err))
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

func (m *Messenger) sendPing(invokerNumber int) error {
	msg := Ping{
		Name: m.name,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("could not marshal ping message: %s", err)
	}

	err = m.provider.WriteMessage(healthTopic, msgBytes)
	if err != nil {
		return fmt.Errorf("could not send ping message: %s", err)
	}

	return nil
}

func (m *Messenger) Close() error {
	return m.provider.Close()
}

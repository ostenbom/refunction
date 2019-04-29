package storage

import (
	"context"
	"fmt"

	"github.com/flimzy/kivik"
	_ "github.com/go-kivik/couchdb" // The CouchDB driver
)

type Activation struct {
	ID           string       `json:"_id"`
	Revision     string       `json:"_rev"`
	ActivationID string       `json:"activationId"`
	Annotations  []Annotation `json:"annotations"`
	Name         string       `json:"name"`
	Namespace    string       `json:"namespace"`
	Response     Response     `json:"response"`
	Start        int          `json:"start"`
	End          int          `json:"end"`
	Subject      string       `json:"subject"`
}

type Response struct {
	Result     interface{} `json:"result"`
	StatusCode int         `json:"statusCode"`
}

type Function struct {
	ID          string        `json:"_id"`
	Revision    string        `json:"_rev"`
	Name        string        `json:"name"`
	Namespace   string        `json:"namespace"`
	Executable  Executable    `json:"exec"`
	Binary      bool          `json:"binary"`
	Limits      Limits        `json:"limits"`
	Parameters  []interface{} `json:"parameters"`
	Annotations []Annotation  `json:"annotations"`
	Publish     bool          `json:"publish"`
	Updated     int           `json:"updated"`
	Version     string        `json:"version"`
}

type Executable struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

type Limits struct {
	Concurrency int `json:"concurrency"`
	Logs        int `json:"logs"`
	Memory      int `json:"memory"`
	Timeout     int `json:"timeout"`
}

type Annotation struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type FunctionStorage interface {
	GetFunction(path string, name string) (*Function, error)
	StoreActivation(Activation) error
}

type functionStorage struct {
	functionDB   *kivik.DB
	activationDB *kivik.DB
}

func NewFunctionStorage(host string, functionDBName string, activationDBName string) (FunctionStorage, error) {
	client, err := kivik.New(context.Background(), "couch", host)
	if err != nil {
		return nil, fmt.Errorf("could not establish connection to database: %s", err)
	}

	functionDB, err := client.DB(context.Background(), functionDBName)
	if err != nil {
		return nil, fmt.Errorf("could not create functionDB connection: %s", err)
	}

	activationDB, err := client.DB(context.Background(), activationDBName)
	if err != nil {
		return nil, fmt.Errorf("could not create activationDB connection: %s", err)
	}

	return functionStorage{
		functionDB:   functionDB,
		activationDB: activationDB,
	}, nil
}

func (s functionStorage) GetFunction(path string, name string) (*Function, error) {
	fullDocID := fmt.Sprintf("%s/%s", path, name)
	row, err := s.functionDB.Get(context.Background(), fullDocID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch database function %s: %s", fullDocID, err)
	}

	var function Function
	err = row.ScanDoc(&function)
	if err != nil {
		return nil, fmt.Errorf("could not parse database function %s: %s", fullDocID, err)
	}
	return &function, nil
}

func (s functionStorage) StoreActivation(activation Activation) error {
	return nil
}

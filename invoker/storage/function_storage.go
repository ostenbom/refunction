package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flimzy/kivik"
	_ "github.com/go-kivik/couchdb" // The CouchDB driver
	"github.com/ostenbom/refunction/invoker/types"
	log "github.com/sirupsen/logrus"
)

type FunctionStorage interface {
	GetFunction(path string, name string) (*types.FunctionDoc, error)
	StoreActivation(*types.ActivationMessage, *types.FunctionDoc, string) error
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

func (s functionStorage) GetFunction(path string, name string) (*types.FunctionDoc, error) {
	fullDocID := fmt.Sprintf("%s/%s", path, name)
	row, err := s.functionDB.Get(context.Background(), fullDocID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch database function %s: %s", fullDocID, err)
	}

	var function types.FunctionDoc
	err = row.ScanDoc(&function)
	if err != nil {
		return nil, fmt.Errorf("could not parse database function %s: %s", fullDocID, err)
	}
	return &function, nil
}

func (s functionStorage) StoreActivation(activationMessage *types.ActivationMessage, function *types.FunctionDoc, result string) error {
	docID := fmt.Sprintf("%s/%s", function.Namespace, activationMessage.ActivationID)

	var resultObject map[string]interface{}
	err := json.Unmarshal([]byte(result), &resultObject)
	if err != nil {
		return fmt.Errorf("could not Unmarshal result to object: %s", err)
	}

	logs := make([]interface{}, 0)
	activation := types.ActivationDoc{
		ID:      docID,
		Updated: int(time.Now().Unix()),
		Response: types.Response{
			ActivationID: activationMessage.ActivationID,
			Annotations:  function.Annotations,
			Name:         function.Name,
			Namespace:    function.Namespace,
			Response: types.ResponseValue{
				Result:     resultObject,
				StatusCode: 0,
			},
			Start:      int(time.Now().Unix()),
			End:        int(time.Now().Unix() + 2),
			Duration:   5,
			Subject:    activationMessage.User.Subject,
			EntityType: "activation",
			Logs:       logs,
			Publish:    function.Publish,
			Version:    function.Version,
		},
	}

	activationsJSON, err := json.Marshal(&activation)
	if err != nil {
		return fmt.Errorf("could not marshal activation: %s", err)
	}
	log.Debug(string(activationsJSON))

	_, err = s.activationDB.Put(context.Background(), docID, activation)
	return err
}

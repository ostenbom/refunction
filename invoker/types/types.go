package types

import (
	"encoding/base64"
	"fmt"
)

type Name struct {
	Instance   int    `json:"instance"`
	UniqueName string `json:"uniqueName"`
	UserMemory string `json:"userMemory"`
}

type ActivationMessage struct {
	Action       Action `json:"action"`
	ActivationID string `json:"activationId"`
	Blocking     bool   `json:"blocking"`
	Controller   struct {
		AsString string `json:"asString"`
	} `json:"rootControllerIndex"`
	TransactionID []interface{} `json:"transid"`
	User          User          `json:"user"`
}

type Action struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type User struct {
	AuthKey struct {
		APIKey string `json:"api_key"`
	} `json:"authKey"`
	Limits    interface{} `json:"limits"`
	Namespace struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	} `json:"namespace"`
	Rights  []interface{} `json:"rights"`
	Subject string        `json:"subject"`
}

type CompletionMessage struct {
	ActivationID  string        `json:"activationId"`
	Invoker       Name          `json:"invoker"`
	SystemError   bool          `json:"isSystemError"`
	TransactionID []interface{} `json:"transid"`
}

type CompletionResponseMessage struct {
	Response      Response      `json:"response"`
	TransactionID []interface{} `json:"transid"`
}

// type Response struct {
// 	ActivationID string               `json:"activationId"`
// 	Annotations  []storage.Annotation `json:"annotations"`
// }

type ActivationDoc struct {
	ID       string `json:"_id"`
	Revision string `json:"_rev,omitempty"`
	Updated  int    `json:"updated"`
	Response
}

type Response struct {
	ActivationID string        `json:"activationId"`
	Annotations  []Annotation  `json:"annotations"`
	Name         string        `json:"name"`
	Namespace    string        `json:"namespace"`
	Response     ResponseValue `json:"response"`
	Start        int           `json:"start"`
	End          int           `json:"end"`
	Duration     int           `json:"duration"`
	Subject      string        `json:"subject"`
	EntityType   string        `json:"entityType"`
	Logs         []interface{} `json:"logs"`
	Publish      bool          `json:"publish"`
	Version      string        `json:"version"`
}

type ResponseValue struct {
	Result     interface{} `json:"result"`
	StatusCode int         `json:"statusCode"`
}

type FunctionDoc struct {
	ID          string        `json:"_id"`
	Revision    string        `json:"_rev"`
	Name        string        `json:"name"`
	Namespace   string        `json:"namespace"`
	Executable  Executable    `json:"exec"`
	Binary      bool          `json:"binary"`
	Limits      Limits        `json:"limits"`
	Parameters  []interface{} `json:"parameters"`
	Annotations []Annotation  `json:"annotations"`
	EntityType  string        `json:"entityType"`
	Publish     bool          `json:"publish"`
	Updated     int           `json:"updated"`
	Version     string        `json:"version"`
}

type Executable struct {
	Kind string      `json:"kind"`
	Code interface{} `json:"code"`
}

func (f *FunctionDoc) CodeString() (string, error) {
	var functionCode string
	switch v := f.Executable.Code.(type) {
	case string:
		functionCode = v
	case map[string]interface{}:
		attachment := v["attachmentName"]
		attachmentString, ok := attachment.(string)
		if !ok {
			return "", fmt.Errorf("attachment code was not string: %s", v)
		}
		b64code := attachmentString[4:] // mem:b64encodedcode
		functionBytes, err := base64.StdEncoding.DecodeString(b64code)
		if err != nil {
			return "", fmt.Errorf("could not decode function %s: %s", v, err)
		}
		functionCode = string(functionBytes)
	default:
		return "", fmt.Errorf("function code not in expected type: %s", f.Executable)
	}
	return functionCode, nil
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

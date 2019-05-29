package types

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type Name struct {
	Instance   int    `json:"instance"`
	UniqueName string `json:"uniqueName"`
	UserMemory string `json:"userMemory"`
}

type ActivationMessage struct {
	Action       Action      `json:"action"`
	ActivationID string      `json:"activationId"`
	Blocking     bool        `json:"blocking"`
	Parameters   interface{} `json:"content"`
	Revision     string      `json:"revision"`
	Controller   struct {
		AsString string `json:"asString"`
	} `json:"rootControllerIndex"`
	TransactionID []interface{} `json:"transid"`
	User          User          `json:"user"`
}

type Action struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
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

func GenerateResponse(activationMessage *ActivationMessage, function *FunctionDoc, result interface{}) Response {
	logs := make([]interface{}, 0)
	return Response{
		ActivationID: activationMessage.ActivationID,
		Annotations:  function.Annotations,
		Name:         function.Name,
		Namespace:    function.Namespace,
		Response: ResponseValue{
			Result:     result,
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
	}
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

		// Java code is a b64 encoded jar
		if f.Executable.Kind == "java" {
			b64url := attachmentString[4:]
			// For some reason when it comes through whisk it is b64url encoded
			b64 := strings.ReplaceAll(strings.ReplaceAll(b64url, "-", "+"), "_", "/")
			return b64, nil
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

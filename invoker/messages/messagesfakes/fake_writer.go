// Code generated by counterfeiter. DO NOT EDIT.
package messagesfakes

import (
	"context"
	"sync"

	"github.com/ostenbom/refunction/invoker/messages"
	kafka "github.com/segmentio/kafka-go"
)

type FakeWriter struct {
	CloseStub        func() error
	closeMutex       sync.RWMutex
	closeArgsForCall []struct {
	}
	closeReturns struct {
		result1 error
	}
	closeReturnsOnCall map[int]struct {
		result1 error
	}
	WriteMessagesStub        func(context.Context, ...kafka.Message) error
	writeMessagesMutex       sync.RWMutex
	writeMessagesArgsForCall []struct {
		arg1 context.Context
		arg2 []kafka.Message
	}
	writeMessagesReturns struct {
		result1 error
	}
	writeMessagesReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeWriter) Close() error {
	fake.closeMutex.Lock()
	ret, specificReturn := fake.closeReturnsOnCall[len(fake.closeArgsForCall)]
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct {
	}{})
	fake.recordInvocation("Close", []interface{}{})
	fake.closeMutex.Unlock()
	if fake.CloseStub != nil {
		return fake.CloseStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.closeReturns
	return fakeReturns.result1
}

func (fake *FakeWriter) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *FakeWriter) CloseCalls(stub func() error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = stub
}

func (fake *FakeWriter) CloseReturns(result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	fake.closeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeWriter) CloseReturnsOnCall(i int, result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	if fake.closeReturnsOnCall == nil {
		fake.closeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.closeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeWriter) WriteMessages(arg1 context.Context, arg2 ...kafka.Message) error {
	fake.writeMessagesMutex.Lock()
	ret, specificReturn := fake.writeMessagesReturnsOnCall[len(fake.writeMessagesArgsForCall)]
	fake.writeMessagesArgsForCall = append(fake.writeMessagesArgsForCall, struct {
		arg1 context.Context
		arg2 []kafka.Message
	}{arg1, arg2})
	fake.recordInvocation("WriteMessages", []interface{}{arg1, arg2})
	fake.writeMessagesMutex.Unlock()
	if fake.WriteMessagesStub != nil {
		return fake.WriteMessagesStub(arg1, arg2...)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.writeMessagesReturns
	return fakeReturns.result1
}

func (fake *FakeWriter) WriteMessagesCallCount() int {
	fake.writeMessagesMutex.RLock()
	defer fake.writeMessagesMutex.RUnlock()
	return len(fake.writeMessagesArgsForCall)
}

func (fake *FakeWriter) WriteMessagesCalls(stub func(context.Context, ...kafka.Message) error) {
	fake.writeMessagesMutex.Lock()
	defer fake.writeMessagesMutex.Unlock()
	fake.WriteMessagesStub = stub
}

func (fake *FakeWriter) WriteMessagesArgsForCall(i int) (context.Context, []kafka.Message) {
	fake.writeMessagesMutex.RLock()
	defer fake.writeMessagesMutex.RUnlock()
	argsForCall := fake.writeMessagesArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeWriter) WriteMessagesReturns(result1 error) {
	fake.writeMessagesMutex.Lock()
	defer fake.writeMessagesMutex.Unlock()
	fake.WriteMessagesStub = nil
	fake.writeMessagesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeWriter) WriteMessagesReturnsOnCall(i int, result1 error) {
	fake.writeMessagesMutex.Lock()
	defer fake.writeMessagesMutex.Unlock()
	fake.WriteMessagesStub = nil
	if fake.writeMessagesReturnsOnCall == nil {
		fake.writeMessagesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.writeMessagesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeWriter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	fake.writeMessagesMutex.RLock()
	defer fake.writeMessagesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeWriter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ messages.Writer = new(FakeWriter)

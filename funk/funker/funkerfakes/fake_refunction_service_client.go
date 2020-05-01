// Code generated by counterfeiter. DO NOT EDIT.
package funkerfakes

import (
	"context"
	"sync"

	refunctionv1alpha "github.com/ostenbom/refunction/cri/service/api/refunction/v1alpha"
	"google.golang.org/grpc"
)

type FakeRefunctionServiceClient struct {
	ListContainersStub        func(context.Context, *refunctionv1alpha.ListContainersRequest, ...grpc.CallOption) (*refunctionv1alpha.ListContainersResponse, error)
	listContainersMutex       sync.RWMutex
	listContainersArgsForCall []struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.ListContainersRequest
		arg3 []grpc.CallOption
	}
	listContainersReturns struct {
		result1 *refunctionv1alpha.ListContainersResponse
		result2 error
	}
	listContainersReturnsOnCall map[int]struct {
		result1 *refunctionv1alpha.ListContainersResponse
		result2 error
	}
	RestoreStub        func(context.Context, *refunctionv1alpha.RestoreRequest, ...grpc.CallOption) (*refunctionv1alpha.RestoreResponse, error)
	restoreMutex       sync.RWMutex
	restoreArgsForCall []struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.RestoreRequest
		arg3 []grpc.CallOption
	}
	restoreReturns struct {
		result1 *refunctionv1alpha.RestoreResponse
		result2 error
	}
	restoreReturnsOnCall map[int]struct {
		result1 *refunctionv1alpha.RestoreResponse
		result2 error
	}
	SendFunctionStub        func(context.Context, *refunctionv1alpha.FunctionRequest, ...grpc.CallOption) (*refunctionv1alpha.FunctionResponse, error)
	sendFunctionMutex       sync.RWMutex
	sendFunctionArgsForCall []struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.FunctionRequest
		arg3 []grpc.CallOption
	}
	sendFunctionReturns struct {
		result1 *refunctionv1alpha.FunctionResponse
		result2 error
	}
	sendFunctionReturnsOnCall map[int]struct {
		result1 *refunctionv1alpha.FunctionResponse
		result2 error
	}
	SendRequestStub        func(context.Context, *refunctionv1alpha.Request, ...grpc.CallOption) (*refunctionv1alpha.Response, error)
	sendRequestMutex       sync.RWMutex
	sendRequestArgsForCall []struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.Request
		arg3 []grpc.CallOption
	}
	sendRequestReturns struct {
		result1 *refunctionv1alpha.Response
		result2 error
	}
	sendRequestReturnsOnCall map[int]struct {
		result1 *refunctionv1alpha.Response
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRefunctionServiceClient) ListContainers(arg1 context.Context, arg2 *refunctionv1alpha.ListContainersRequest, arg3 ...grpc.CallOption) (*refunctionv1alpha.ListContainersResponse, error) {
	fake.listContainersMutex.Lock()
	ret, specificReturn := fake.listContainersReturnsOnCall[len(fake.listContainersArgsForCall)]
	fake.listContainersArgsForCall = append(fake.listContainersArgsForCall, struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.ListContainersRequest
		arg3 []grpc.CallOption
	}{arg1, arg2, arg3})
	fake.recordInvocation("ListContainers", []interface{}{arg1, arg2, arg3})
	fake.listContainersMutex.Unlock()
	if fake.ListContainersStub != nil {
		return fake.ListContainersStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.listContainersReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeRefunctionServiceClient) ListContainersCallCount() int {
	fake.listContainersMutex.RLock()
	defer fake.listContainersMutex.RUnlock()
	return len(fake.listContainersArgsForCall)
}

func (fake *FakeRefunctionServiceClient) ListContainersCalls(stub func(context.Context, *refunctionv1alpha.ListContainersRequest, ...grpc.CallOption) (*refunctionv1alpha.ListContainersResponse, error)) {
	fake.listContainersMutex.Lock()
	defer fake.listContainersMutex.Unlock()
	fake.ListContainersStub = stub
}

func (fake *FakeRefunctionServiceClient) ListContainersArgsForCall(i int) (context.Context, *refunctionv1alpha.ListContainersRequest, []grpc.CallOption) {
	fake.listContainersMutex.RLock()
	defer fake.listContainersMutex.RUnlock()
	argsForCall := fake.listContainersArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeRefunctionServiceClient) ListContainersReturns(result1 *refunctionv1alpha.ListContainersResponse, result2 error) {
	fake.listContainersMutex.Lock()
	defer fake.listContainersMutex.Unlock()
	fake.ListContainersStub = nil
	fake.listContainersReturns = struct {
		result1 *refunctionv1alpha.ListContainersResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) ListContainersReturnsOnCall(i int, result1 *refunctionv1alpha.ListContainersResponse, result2 error) {
	fake.listContainersMutex.Lock()
	defer fake.listContainersMutex.Unlock()
	fake.ListContainersStub = nil
	if fake.listContainersReturnsOnCall == nil {
		fake.listContainersReturnsOnCall = make(map[int]struct {
			result1 *refunctionv1alpha.ListContainersResponse
			result2 error
		})
	}
	fake.listContainersReturnsOnCall[i] = struct {
		result1 *refunctionv1alpha.ListContainersResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) Restore(arg1 context.Context, arg2 *refunctionv1alpha.RestoreRequest, arg3 ...grpc.CallOption) (*refunctionv1alpha.RestoreResponse, error) {
	fake.restoreMutex.Lock()
	ret, specificReturn := fake.restoreReturnsOnCall[len(fake.restoreArgsForCall)]
	fake.restoreArgsForCall = append(fake.restoreArgsForCall, struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.RestoreRequest
		arg3 []grpc.CallOption
	}{arg1, arg2, arg3})
	fake.recordInvocation("Restore", []interface{}{arg1, arg2, arg3})
	fake.restoreMutex.Unlock()
	if fake.RestoreStub != nil {
		return fake.RestoreStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.restoreReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeRefunctionServiceClient) RestoreCallCount() int {
	fake.restoreMutex.RLock()
	defer fake.restoreMutex.RUnlock()
	return len(fake.restoreArgsForCall)
}

func (fake *FakeRefunctionServiceClient) RestoreCalls(stub func(context.Context, *refunctionv1alpha.RestoreRequest, ...grpc.CallOption) (*refunctionv1alpha.RestoreResponse, error)) {
	fake.restoreMutex.Lock()
	defer fake.restoreMutex.Unlock()
	fake.RestoreStub = stub
}

func (fake *FakeRefunctionServiceClient) RestoreArgsForCall(i int) (context.Context, *refunctionv1alpha.RestoreRequest, []grpc.CallOption) {
	fake.restoreMutex.RLock()
	defer fake.restoreMutex.RUnlock()
	argsForCall := fake.restoreArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeRefunctionServiceClient) RestoreReturns(result1 *refunctionv1alpha.RestoreResponse, result2 error) {
	fake.restoreMutex.Lock()
	defer fake.restoreMutex.Unlock()
	fake.RestoreStub = nil
	fake.restoreReturns = struct {
		result1 *refunctionv1alpha.RestoreResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) RestoreReturnsOnCall(i int, result1 *refunctionv1alpha.RestoreResponse, result2 error) {
	fake.restoreMutex.Lock()
	defer fake.restoreMutex.Unlock()
	fake.RestoreStub = nil
	if fake.restoreReturnsOnCall == nil {
		fake.restoreReturnsOnCall = make(map[int]struct {
			result1 *refunctionv1alpha.RestoreResponse
			result2 error
		})
	}
	fake.restoreReturnsOnCall[i] = struct {
		result1 *refunctionv1alpha.RestoreResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) SendFunction(arg1 context.Context, arg2 *refunctionv1alpha.FunctionRequest, arg3 ...grpc.CallOption) (*refunctionv1alpha.FunctionResponse, error) {
	fake.sendFunctionMutex.Lock()
	ret, specificReturn := fake.sendFunctionReturnsOnCall[len(fake.sendFunctionArgsForCall)]
	fake.sendFunctionArgsForCall = append(fake.sendFunctionArgsForCall, struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.FunctionRequest
		arg3 []grpc.CallOption
	}{arg1, arg2, arg3})
	fake.recordInvocation("SendFunction", []interface{}{arg1, arg2, arg3})
	fake.sendFunctionMutex.Unlock()
	if fake.SendFunctionStub != nil {
		return fake.SendFunctionStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.sendFunctionReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeRefunctionServiceClient) SendFunctionCallCount() int {
	fake.sendFunctionMutex.RLock()
	defer fake.sendFunctionMutex.RUnlock()
	return len(fake.sendFunctionArgsForCall)
}

func (fake *FakeRefunctionServiceClient) SendFunctionCalls(stub func(context.Context, *refunctionv1alpha.FunctionRequest, ...grpc.CallOption) (*refunctionv1alpha.FunctionResponse, error)) {
	fake.sendFunctionMutex.Lock()
	defer fake.sendFunctionMutex.Unlock()
	fake.SendFunctionStub = stub
}

func (fake *FakeRefunctionServiceClient) SendFunctionArgsForCall(i int) (context.Context, *refunctionv1alpha.FunctionRequest, []grpc.CallOption) {
	fake.sendFunctionMutex.RLock()
	defer fake.sendFunctionMutex.RUnlock()
	argsForCall := fake.sendFunctionArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeRefunctionServiceClient) SendFunctionReturns(result1 *refunctionv1alpha.FunctionResponse, result2 error) {
	fake.sendFunctionMutex.Lock()
	defer fake.sendFunctionMutex.Unlock()
	fake.SendFunctionStub = nil
	fake.sendFunctionReturns = struct {
		result1 *refunctionv1alpha.FunctionResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) SendFunctionReturnsOnCall(i int, result1 *refunctionv1alpha.FunctionResponse, result2 error) {
	fake.sendFunctionMutex.Lock()
	defer fake.sendFunctionMutex.Unlock()
	fake.SendFunctionStub = nil
	if fake.sendFunctionReturnsOnCall == nil {
		fake.sendFunctionReturnsOnCall = make(map[int]struct {
			result1 *refunctionv1alpha.FunctionResponse
			result2 error
		})
	}
	fake.sendFunctionReturnsOnCall[i] = struct {
		result1 *refunctionv1alpha.FunctionResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) SendRequest(arg1 context.Context, arg2 *refunctionv1alpha.Request, arg3 ...grpc.CallOption) (*refunctionv1alpha.Response, error) {
	fake.sendRequestMutex.Lock()
	ret, specificReturn := fake.sendRequestReturnsOnCall[len(fake.sendRequestArgsForCall)]
	fake.sendRequestArgsForCall = append(fake.sendRequestArgsForCall, struct {
		arg1 context.Context
		arg2 *refunctionv1alpha.Request
		arg3 []grpc.CallOption
	}{arg1, arg2, arg3})
	fake.recordInvocation("SendRequest", []interface{}{arg1, arg2, arg3})
	fake.sendRequestMutex.Unlock()
	if fake.SendRequestStub != nil {
		return fake.SendRequestStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.sendRequestReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeRefunctionServiceClient) SendRequestCallCount() int {
	fake.sendRequestMutex.RLock()
	defer fake.sendRequestMutex.RUnlock()
	return len(fake.sendRequestArgsForCall)
}

func (fake *FakeRefunctionServiceClient) SendRequestCalls(stub func(context.Context, *refunctionv1alpha.Request, ...grpc.CallOption) (*refunctionv1alpha.Response, error)) {
	fake.sendRequestMutex.Lock()
	defer fake.sendRequestMutex.Unlock()
	fake.SendRequestStub = stub
}

func (fake *FakeRefunctionServiceClient) SendRequestArgsForCall(i int) (context.Context, *refunctionv1alpha.Request, []grpc.CallOption) {
	fake.sendRequestMutex.RLock()
	defer fake.sendRequestMutex.RUnlock()
	argsForCall := fake.sendRequestArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeRefunctionServiceClient) SendRequestReturns(result1 *refunctionv1alpha.Response, result2 error) {
	fake.sendRequestMutex.Lock()
	defer fake.sendRequestMutex.Unlock()
	fake.SendRequestStub = nil
	fake.sendRequestReturns = struct {
		result1 *refunctionv1alpha.Response
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) SendRequestReturnsOnCall(i int, result1 *refunctionv1alpha.Response, result2 error) {
	fake.sendRequestMutex.Lock()
	defer fake.sendRequestMutex.Unlock()
	fake.SendRequestStub = nil
	if fake.sendRequestReturnsOnCall == nil {
		fake.sendRequestReturnsOnCall = make(map[int]struct {
			result1 *refunctionv1alpha.Response
			result2 error
		})
	}
	fake.sendRequestReturnsOnCall[i] = struct {
		result1 *refunctionv1alpha.Response
		result2 error
	}{result1, result2}
}

func (fake *FakeRefunctionServiceClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.listContainersMutex.RLock()
	defer fake.listContainersMutex.RUnlock()
	fake.restoreMutex.RLock()
	defer fake.restoreMutex.RUnlock()
	fake.sendFunctionMutex.RLock()
	defer fake.sendFunctionMutex.RUnlock()
	fake.sendRequestMutex.RLock()
	defer fake.sendRequestMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeRefunctionServiceClient) recordInvocation(key string, args []interface{}) {
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

var _ refunctionv1alpha.RefunctionServiceClient = new(FakeRefunctionServiceClient)

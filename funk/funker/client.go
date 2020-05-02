package funker

import (
	"context"
	"io/ioutil"
	"os"
	"path"

	refunction "github.com/ostenbom/refunction/cri/service/api/refunction/v1alpha"
	"google.golang.org/grpc"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Client

type Client interface {
	refunction.RefunctionServiceClient
	Start() error
	Close()
}

type client struct {
	refunctionClient refunction.RefunctionServiceClient
	conn             *grpc.ClientConn
}

func (c *client) Start() error {
	target, err := getTarget()
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return err
	}
	c.conn = conn

	c.refunctionClient = refunction.NewRefunctionServiceClient(conn)

	return nil
}

func (c *client) Close() {
	c.conn.Close()
}

func (c *client) ListContainers(ctx context.Context, in *refunction.ListContainersRequest, opts ...grpc.CallOption) (*refunction.ListContainersResponse, error) {
	return c.refunctionClient.ListContainers(ctx, in, opts...)
}

func (c *client) SendRequest(ctx context.Context, in *refunction.Request, opts ...grpc.CallOption) (*refunction.Response, error) {
	return c.refunctionClient.SendRequest(ctx, in, opts...)
}

func (c *client) SendFunction(ctx context.Context, in *refunction.FunctionRequest, opts ...grpc.CallOption) (*refunction.FunctionResponse, error) {
	return c.refunctionClient.SendFunction(ctx, in, opts...)
}

func (c *client) Restore(ctx context.Context, in *refunction.RestoreRequest, opts ...grpc.CallOption) (*refunction.RestoreResponse, error) {
	return c.refunctionClient.Restore(ctx, in, opts...)
}

func getTarget() (string, error) {
	targetBytes, err := ioutil.ReadFile(path.Join(os.Getenv("HOME"), ".funkrc"))
	if err != nil {
		return "", err
	}

	return string(targetBytes), nil
}

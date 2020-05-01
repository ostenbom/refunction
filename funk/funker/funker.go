package funker

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	refunction "github.com/ostenbom/refunction/cri/service/api/refunction/v1alpha"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 github.com/ostenbom/refunction/cri/service/api/refunction/v1alpha.RefunctionServiceClient

type Funker struct {
	client refunction.RefunctionServiceClient
}

func (f *Funker) Start() error {
	if f.client != nil {
		return nil
	}

	target, err := getTarget()
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return err
	}

	f.client = refunction.NewRefunctionServiceClient(conn)

	return nil
}

func NewFakeFunker(client refunction.RefunctionServiceClient) *Funker {
	return &Funker{
		client: client,
	}
}

func NewFunker() *Funker {
	return &Funker{}
}

func (f *Funker) App() *cli.App {
	return &cli.App{
		Name:  "funk",
		Usage: "communicate with k8s refunction containers",
		Commands: []*cli.Command{
			{
				Name:    "target",
				Aliases: []string{"t"},
				Usage:   "save target kubelet url",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "url", Aliases: []string{"u"}, Required: true},
				},
				Action: f.SaveTarget,
			},
			{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "list refunction containers",
				Action:  f.ListContainers,
			},
			{
				Name:    "function",
				Aliases: []string{"f"},
				Usage:   "load function into container",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "container",
						Aliases:  []string{"c"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "function",
						Aliases:  []string{"f"},
						Required: true,
					},
				},
				Action: f.SendFunction,
			},
			{
				Name:    "request",
				Aliases: []string{"r"},
				Usage:   "send request to container",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "container",
						Aliases:  []string{"c"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "request",
						Aliases:  []string{"r"},
						Required: true,
					},
				},
				Action: f.SendFunction,
			},
		},
	}
}

func (f *Funker) SaveTarget(c *cli.Context) error {
	fmt.Printf("saving target: %s\n", c.String("url"))
	return ioutil.WriteFile(path.Join(os.Getenv("HOME"), ".funkrc"), []byte(c.String("url")), os.ModePerm)
}

func (f *Funker) ListContainers(c *cli.Context) error {
	err := f.Start()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listResponse, err := f.client.ListContainers(ctx, &refunction.ListContainersRequest{})
	if err != nil {
		return err
	}

	fmt.Println("CONTAINERS")

	for _, id := range listResponse.ContainerIds {
		fmt.Println(id)
	}

	return nil
}

func (f *Funker) SendFunction(c *cli.Context) error {
	err := f.Start()
	if err != nil {
		return err
	}

	container := c.String("container")
	function := c.String("function")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = f.client.SendFunction(ctx, &refunction.FunctionRequest{
		Function:    function,
		ContainerId: container,
	})
	if err != nil {
		return err
	}

	fmt.Printf("sent function %s to container %s\n", function, container)

	return nil
}

func (f *Funker) SendRequest(c *cli.Context) error {
	err := f.Start()
	if err != nil {
		return err
	}

	container := c.String("container")
	request := c.String("request")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	requestResponse, err := f.client.SendRequest(ctx, &refunction.Request{
		Request:     request,
		ContainerId: container,
	})
	if err != nil {
		return err
	}

	fmt.Printf("sent request %s to container %s\n", request, container)
	fmt.Printf("response: %s\n", requestResponse.Response)

	return nil
}

func getTarget() (string, error) {
	targetBytes, err := ioutil.ReadFile(path.Join(os.Getenv("HOME"), ".funkrc"))
	if err != nil {
		return "", err
	}

	return string(targetBytes), nil
}

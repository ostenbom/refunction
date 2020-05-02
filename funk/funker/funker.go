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
)

type Funker struct {
	client Client
}

func (f *Funker) Start() error {
	return f.client.Start()
}

func (f *Funker) Close() {
	f.client.Close()
}

func NewFakeFunker(client Client) *Funker {
	return &Funker{
		client: client,
	}
}

func NewFunker() *Funker {
	return &Funker{
		client: &client{},
	}
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
				Action: f.SendRequest,
			},
			{
				Name:  "restore",
				Usage: "restore container to initial state",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "container",
						Aliases:  []string{"c"},
						Required: true,
					},
				},
				Action: f.Restore,
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

	f.Close()
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

	f.Close()
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

	f.Close()
	return nil
}

func (f *Funker) Restore(c *cli.Context) error {
	err := f.Start()
	if err != nil {
		return err
	}

	container := c.String("container")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = f.client.Restore(ctx, &refunction.RestoreRequest{
		ContainerId: container,
	})
	if err != nil {
		return err
	}

	fmt.Printf("restored container %s\n", container)

	f.Close()
	return nil
}

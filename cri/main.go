package main

import (
	"log"
	"net"
	"os"

	"github.com/containerd/containerd"
	"github.com/ostenbom/refunction/cri/service"
	"google.golang.org/grpc"
)

var GRPCSocketAddr = "/tmp/refunction.sock"
var containerdSocketAddr = "/run/containerd/containerd.sock"
var K8sContainerdNamespace = "k8s.io"
var refunctionServerAddress = "localhost:7777"

func startCRIService() int {
	if err := os.RemoveAll(GRPCSocketAddr); err != nil {
		log.Fatalf("could not remove socket: %v", err)
		return 1
	}

	_, err := os.Stat(containerdSocketAddr)
	if err != nil {
		log.Fatalf("containerd isn't running or socket does not exist: %v\n", err)
		return 1
	}

	criLis, err := net.Listen("unix", GRPCSocketAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return 1
	}

	client, err := containerd.New(containerdSocketAddr, containerd.WithDefaultNamespace(K8sContainerdNamespace))
	if err != nil {
		log.Fatalf("could not connect to containerd client: %s", err)
		return 1
	}

	criService, err := service.NewCRIService(client)
	if err != nil {
		log.Fatalf("could not start CRI: %s", err)
		return 1
	}

	refunctionLis, err := net.Listen("tcp", refunctionServerAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return 1
	}

	criServer := grpc.NewServer()
	refunctionServer := grpc.NewServer()

	criService.Register(criServer, refunctionServer)

	err = criServer.Serve(criLis)
	if err != nil {
		log.Fatalf("could not start grpcServer: %v", err)
		return 1
	}

	err = refunctionServer.Serve(refunctionLis)
	if err != nil {
		log.Fatalf("could not start grpcServer: %v", err)
		return 1
	}

	return 0
}

func main() {
	exitCode := startCRIService()
	os.Exit(exitCode)
}

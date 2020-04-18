package main

import (
	"log"
	"net"
	"os"

	"github.com/containerd/containerd"
	"google.golang.org/grpc"
)

var GRPCSocketAddr = "/tmp/refunction.sock"
var containerdSocketAddr = "/run/containerd/containerd.sock"

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

	lis, err := net.Listen("unix", GRPCSocketAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return 1
	}

	client, err := containerd.New(containerdSocketAddr)
	if err != nil {
		log.Fatalf("could not connect to containerd client: %s", err)
		return 1
	}

	criService, err := NewCRIService(client)
	if err != nil {
		log.Fatalf("could not start CRI: %s", err)
		return 1
	}

	grpcServer := grpc.NewServer()
	criService.register(grpcServer)

	err = grpcServer.Serve(lis)
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

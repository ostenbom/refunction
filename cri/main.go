package main

import "net"
import "os"
import "log"
import "google.golang.org/grpc"

var SocketAddr = "/tmp/refunction.sock"

func main() {
	if err := os.RemoveAll(SocketAddr); err != nil {
		log.Fatalf("could not remove socket: %v", err)
	}

	lis, err := net.Listen("unix", SocketAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	criService := NewCRIService()
	criService.register(grpcServer)
	grpcServer.Serve(lis)
}

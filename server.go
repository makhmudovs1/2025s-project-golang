package main

import (
	blogpb "blog/blog/gen"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	// Слушаем порт 50051 (стандартный для gRPC)
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	blogpb.RegisterBlogServiceServer(grpcServer, &BlogServer{})
	log.Println("Server listening on port 50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}

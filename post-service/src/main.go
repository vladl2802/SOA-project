package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "soa-project/post-service/proto"
)

func main() {
	grpcAddr := os.Getenv("GRPC_ADDR")
	databaseUrl := os.Getenv("DATABASE_ADDR")

	log.Printf("GRPC_ADDR: `%v`\n", grpcAddr)
	log.Printf("DATABASE_ADDR: `%v`\n", databaseUrl)

	if grpcAddr == "" {
		log.Fatal("grpc address not provided")
	}

	if databaseUrl == "" {
		log.Fatal("database url not provided")
	}

	postService, err := NewUserService(databaseUrl)
	if err != nil {
		log.Fatalf("failed to initialize PostService: %v", err)
	}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterPostServiceServer(grpcServer, postService)
	grpcServer.Serve(lis)
}

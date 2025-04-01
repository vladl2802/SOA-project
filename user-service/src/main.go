package main

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"google.golang.org/grpc"

	pb "soa-project/user-service/proto"
)

func main() {
	jwtPrivateFile := os.Getenv("JWT_PRIVATE")
	grpcAddr := os.Getenv("GRPC_ADDR")
	databaseUrl := os.Getenv("DATABASE_ADDR")

	log.Printf("GRPC_ADDR: `%v`\n", grpcAddr)
	log.Printf("DATABASE_ADDR: `%v`\n", databaseUrl)

	if jwtPrivateFile == "" {
		log.Fatalf("jwt private key file not provided")
	}

	absolutePrivateFile, err := filepath.Abs(jwtPrivateFile)
	if err != nil {
		log.Fatalf("failed to obtain absolute path to private file: %v", err)
	}

	if grpcAddr == "" {
		log.Fatal("grpc address not provided")
	}

	if databaseUrl == "" {
		log.Fatal("database url not provided")
	}

	userService, err := NewUserService(absolutePrivateFile, databaseUrl)
	if err != nil {
		log.Fatalf("failed to initialize UserService: %v", err)
	}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, userService)
	grpcServer.Serve(lis)
}

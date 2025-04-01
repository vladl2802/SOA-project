package main

import (
	"log"
	"os"
	"path/filepath"
	"soa-project/api-service/handles"
	postservice "soa-project/post-service/proto"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userservice "soa-project/user-service/proto"
)

func main() {
	jwtPublicFile := os.Getenv("JWT_PUBLIC")
	userserviceGrpcAddr := os.Getenv("USERSERVICE_GRPC_ADDR")
	postserviceGrpcAddr := os.Getenv("POSTSERVICE_GRPC_ADDR")

	log.Printf("USERSERVICE_GRPC_ADDR: %v", userserviceGrpcAddr)
	log.Printf("POSTSERVICE_GRPC_ADDR: %v", postserviceGrpcAddr)

	if jwtPublicFile == "" {
		log.Fatalf("jwt public key file not provided")
	}

	absolutePublicFile, err := filepath.Abs(jwtPublicFile)
	if err != nil {
		log.Fatalf("failed to obtain absolute path to public file: %v", err)
	}
	public, err := os.ReadFile(absolutePublicFile)
	if err != nil {
		log.Fatalf("failed to read jwtPublicFile: %v", err)
	}
	jwtPublic, err := jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		log.Fatalf("failed to parse public key: %v", err)
	}

	if userserviceGrpcAddr == "" {
		log.Fatalf("userservice grpc address not provided")
	}
	if postserviceGrpcAddr == "" {
		log.Fatalf("postservice grpc address not provided")
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	userserviceConn, err := grpc.NewClient(userserviceGrpcAddr, opts...)
	if err != nil {
		log.Fatalf("failed to create grpc connection with userservice: %v", err)
	}

	postserviceConn, err := grpc.NewClient(postserviceGrpcAddr, opts...)
	if err != nil {
		log.Fatalf("failed to create grpc connection with postservice: %v", err)
	}

	handleContext := handles.HandleContext{
		UserserviceClient: userservice.NewUserServiceClient(userserviceConn),
		PostserviceClient: postservice.NewPostServiceClient(postserviceConn),
		JwtPublic:         jwtPublic,
	}

	engine := gin.Default()
	handleContext.HandleUserService(engine)
	handleContext.HandlePostService(engine)

	engine.Run()
}

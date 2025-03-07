ROOT= ../../

compile_proto:
	protoc --proto_path=./proto --proto_path=${ROOT}/shared/proto --go_out=./proto --go_opt=paths=source_relative \
		--go-grpc_out=./proto --go-grpc_opt=paths=source_relative \
		./proto/user-service.proto

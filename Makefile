.PHONY: protoc cert client server server-tls client client-tls

protoc:
	protoc --go_out=. --go-grpc_out=. streaming/streaming.proto

cert:
	@cd cert; ./gen.sh; cd ..

server:
	@go run ./cmd/server/main.go -port=50051

server-tls:
	@go run ./cmd/server/main.go -port=50051 -tls

client:
	@go run ./cmd/client/main.go -address=0.0.0.0:50051

client-tls:
	@go run ./cmd/client/main.go -address=0.0.0.0:50051 -tls
.PHONY: lint protoc cert client server server-tls server-mutual-tls client client-tls client-mutual-tls

lint:
	golangci-lint run

protoc:
	protoc --go_out=. --go-grpc_out=. streaming/streaming.proto

cert:
	@cd cert; ./gen.sh; cd ..

server:
	@go run ./cmd/server/main.go -port=50051

server-tls:
	@go run ./cmd/server/main.go -port=50051 -tls

server-mutual-tls:
	@go run ./cmd/server/main.go -port=50051 -tls -mutualTLS

client:
	@go run ./cmd/client/main.go -address=0.0.0.0:50051

client-tls:
	@go run ./cmd/client/main.go -address=0.0.0.0:50051 -tls

client-mutual-tls:
	@go run ./cmd/client/main.go -address=0.0.0.0:50051 -tls -mutualTLS
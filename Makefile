.PHONY: lint protoc cert client server server-tls server-mutual-tls client client-tls client-mutual-tls

.DEFAULT_GOAL := help

lint:
	golangci-lint runmake

protoc:
	protoc --go_out=. --go-grpc_out=. streaming/streaming.proto

cert:
	@cd cert; ./gen.sh; cd ..

server: ## run server without tls
	@go run ./cmd/server/main.go -port=50051

server-tls: ## run server with tls
	@go run ./cmd/server/main.go -port=50051 -tls

server-mutual-tls: ## run server with mutual tls
	@go run ./cmd/server/main.go -port=50051 -tls -mutualTLS

client: ## run client without tls
	@go run ./cmd/client/main.go -address=0.0.0.0:50051

client-tls: ## run client with tls
	@go run ./cmd/client/main.go -address=0.0.0.0:50051 -tls


client-mutual-tls: ## run client with mutual tls
	@go run ./cmd/client/main.go -address=0.0.0.0:50051 -tls -mutualTLS

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

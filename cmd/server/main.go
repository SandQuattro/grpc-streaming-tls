package main

import (
	"flag"
	"grpc-streaming/internal/server/interceptors"
	creds "grpc-streaming/internal/server/tls"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"
	pb "grpc-streaming/streaming/grpc"
)

type server struct {
	pb.UnimplementedChatServer
}

func (s *server) ChatStream(stream pb.Chat_ChatStreamServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			slog.Warn("client streaming finished")
			// Если клиент завершил отправку
			return nil
		}
		if err != nil {
			slog.With("error", err).Error("[ERROR] client finished with error")
			return err
		}

		slog.With("body", msg.Body).Info("Received message body from client")

		// Отправляем сообщение обратно клиенту
		if err := stream.Send(&pb.Message{Body: msg.Body}); err != nil {
			return err
		}
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	port := flag.Int("port", 0, "the server port")
	enableTLS := flag.Bool("tls", false, "enable SSL/TLS")
	mutualTLS := flag.Bool("mutualTLS", false, "enable client certificate verification")

	flag.Parse()
	logger.With("port", *port, "TLS", *enableTLS, "mutualTLS", *mutualTLS).Info("started server")

	interceptor := interceptors.NewAuthServerInterceptor([]string{"user"})
	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
	}

	if *enableTLS {
		tlsCredentials, err := creds.LoadServerTLSCredentials(*mutualTLS)
		if err != nil {
			logger.With("error", err).Error("cannot load TLS credentials")
			os.Exit(1)
		}

		serverOptions = append(serverOptions, grpc.Creds(tlsCredentials))
	}

	grpcServer := grpc.NewServer(serverOptions...)

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		logger.With("error", err).Error("failed to listen tcp port")
		os.Exit(1)
	}

	pb.RegisterChatServer(grpcServer, &server{})
	if err = grpcServer.Serve(lis); err != nil {
		logger.With("error", err).Error("failed to serve grpc")
		os.Exit(1)
	}
}

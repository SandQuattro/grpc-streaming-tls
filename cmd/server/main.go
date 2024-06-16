package main

import (
	"crypto/tls"
	"flag"
	"google.golang.org/grpc/credentials"
	"grpc-streaming/internal/server/interceptors"
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
			slog.Warn("client finished")
			// Если клиент завершил отправку
			return nil
		}
		if err != nil {
			slog.Error("client finished with error: %v", err)
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

	flag.Parse()
	logger.With("port", *port, "TLS", *enableTLS).Info("started server")

	interceptor := interceptors.NewAuthServerInterceptor([]string{"user"})
	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
	}

	if *enableTLS {
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			logger.Error("cannot load TLS credentials: ", err)
			os.Exit(1)
		}

		serverOptions = append(serverOptions, grpc.Creds(tlsCredentials))
	}

	grpcServer := grpc.NewServer(serverOptions...)

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		logger.Error("failed to listen: %v", err)
		os.Exit(1)
	}

	pb.RegisterChatServer(grpcServer, &server{})
	if err = grpcServer.Serve(lis); err != nil {
		logger.Error("failed to serve: %v", err)
		os.Exit(1)
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}

	return credentials.NewTLS(config), nil
}

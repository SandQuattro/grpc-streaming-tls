package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/brianvoe/gofakeit/v7"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"grpc-streaming/internal/client/interceptors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	pb "grpc-streaming/streaming/grpc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	address := flag.String("address", "", "the server address")
	enableTLS := flag.Bool("tls", false, "enable SSL/TLS")

	flag.Parse()
	logger.With("address", *address, "TLS", *enableTLS).Info("connecting to server...")

	parentCtx, cancel := context.WithCancel(context.Background())

	interceptor := interceptors.NewAuthClientInterceptor()
	clientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	}

	if *enableTLS {
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			logger.Error("cannot load TLS credentials: ", err)
			os.Exit(1)
		}

		clientOptions = append(clientOptions, grpc.WithTransportCredentials(tlsCredentials))
	}

	conn, err := grpc.NewClient(*address, clientOptions...)
	if err != nil {
		logger.Error("did not connect: %v", err)
		return
	}
	defer conn.Close()

	client := pb.NewChatClient(conn)

	stream, err := client.ChatStream(parentCtx)
	if err != nil {
		logger.With(slog.String("error", err.Error())).Error("Error creating stream")
		return
	}

	// Горутина получения сообщений
	go func() {
		defer cancel()
		for {
			select {
			case <-parentCtx.Done():
				logger.Debug("[RECEIVER] Received signal, shutting down")
				return
			default:
				in, err := stream.Recv()

				// Ловим отмену контекста и переходим на обработку канала done в select
				if status.Code(err) == codes.Canceled {
					continue
				}

				if err == io.EOF {
					// Сервер прекратил отправку или контекст завершен
					return
				}

				if err != nil {
					logger.With(slog.String("error", err.Error())).Error("Failed to receive a message")
					return
				}
				logger.With("body", in.Body).Debug("got server message")
			}
		}
	}()

	// Горутина отправки сообщений
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-parentCtx.Done():
				logger.Debug("[SENDER] Received signal, shutting down")
				return
			case <-ticker.C:
				msg := gofakeit.Name() + " want to drink " + gofakeit.BeerName()
				if err = stream.Send(&pb.Message{Body: msg}); err != nil {
					logger.With(slog.String("error", err.Error())).Error("Failed to send a message")
				}
			}
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)

	go func() {
		<-stop
		logger.Debug("shutting down...")
		err = stream.CloseSend()
		if err != nil {
			logger.With(slog.String("error", err.Error())).Error("Failed to close stream")
			return
		}
		logger.Debug("Closed stream")
		cancel()
	}()

	<-parentCtx.Done()
	logger.Warn("Bye!")
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := os.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return credentials.NewTLS(config), nil
}

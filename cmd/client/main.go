package main

import (
	"context"
	"errors"
	"flag"
	"github.com/brianvoe/gofakeit/v7"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"grpc-streaming/internal/client/interceptors"
	creds "grpc-streaming/internal/client/tls"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	pb "grpc-streaming/streaming/grpc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	var address string
	var enableTLS, mutualTLS bool

	flag.StringVar(&address, "address", "", "the server address")
	flag.BoolVar(&enableTLS, "tls", false, "enable SSL/TLS")
	flag.BoolVar(&mutualTLS, "mutualTLS", false, "enable mutual TLS")
	flag.Parse()

	logger.With("address", address, "TLS", enableTLS, "mutualTLS", mutualTLS).Info("connecting to server...")

	parentCtx, cancel := context.WithCancel(context.Background())

	interceptor := interceptors.NewAuthClientInterceptor()
	clientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	}

	if enableTLS {
		tlsCredentials, err := creds.LoadClientTLSCredentials(mutualTLS)
		if err != nil {
			logger.With("error", err).Error("cannot load client TLS credentials")
			os.Exit(1)
		}

		clientOptions = append(clientOptions, grpc.WithTransportCredentials(tlsCredentials))
	}

	conn, err := grpc.NewClient(address, clientOptions...)
	if err != nil {
		logger.With("error", err).Error("[ERROR] grpc client did not connect")
		return
	}
	defer conn.Close()

	client := pb.NewChatClient(conn)

	stream, err := client.ChatStream(parentCtx)
	if err != nil {
		logger.With(slog.String("error", err.Error())).Error("[ERROR] creating stream failed")
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

				if errors.Is(err, io.EOF) {
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
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

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

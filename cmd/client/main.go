package main

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"io"
	slog "log/slog"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	pb "grpc-streaming/streaming/grpc"
)

func main() {
	parentCtx, cancel := context.WithCancel(context.Background())

	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

	logger := slog.New(h)

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
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

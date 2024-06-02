package main

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	slog "log/slog"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	pb "grpc-streaming/streaming/grpc"
)

func main() {
	ctx := context.Background()
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(h)

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("did not connect: %v", err)
		return
	}
	defer conn.Close()

	client := pb.NewChatClient(conn)

	stream, err := client.ChatStream(ctx)
	if err != nil {
		logger.With(slog.String("error", err.Error())).Error("Error creating stream")
		return
	}

	notifyContext, stop := signal.NotifyContext(ctx, os.Interrupt)

	// Горутина получения сообщений
	go func() {
		defer stop()
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// Сервер прекратил отправку
				return
			}
			if err != nil {
				logger.With(slog.String("error", err.Error())).Error("Failed to receive a message")
				return
			}
			logger.With("body", in.Body).Debug("got server message")
		}
	}()

	// Горутина отправки сообщений
	go func() {
		for {
			select {
			case <-notifyContext.Done():
				logger.Warn("Received signal, shutting down")
				return
			default:
				msg := gofakeit.Name() + " sending message " + gofakeit.BeerName()
				if err = stream.Send(&pb.Message{Body: msg}); err != nil {
					logger.With(slog.String("error", err.Error())).Error("Failed to send a message")
				}
				time.Sleep(time.Second)
			}
		}
	}()

	<-notifyContext.Done()

	logger.Debug("shutting down...")
	// Завершение работы, плавно тушим stream
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	go func() {
		err = stream.CloseSend()
		if err != nil {
			logger.With(slog.String("error", err.Error())).Error("Failed to close stream")
			return
		}
		logger.Debug("Closed stream")
		cancelFunc()
	}()

	<-ctx.Done()
}

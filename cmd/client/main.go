package main

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	pb "grpc-streaming/streaming/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewChatClient(conn)
	ctx := context.Background()

	stream, err := client.ChatStream(ctx)
	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}

	waitc := make(chan os.Signal, 1)

	// Горутин для получения сообщений
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// Сервер прекратил отправку
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a message : %v", err)
			}
			log.Printf("Got message: %s", in.Body)
		}
	}()

	go func() {
		for {
			msg := gofakeit.Name() + " sending message " + gofakeit.BeerName()

			if err = stream.Send(&pb.Message{Body: msg}); err != nil {
				log.Fatalf("Failed to send a message: %v", err)
			}

			time.Sleep(time.Second)
		}
	}()

	signal.Notify(waitc, os.Interrupt, os.Kill)

	<-waitc

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	go func() {
		stream.CloseSend()
		cancelFunc()
	}()

	<-ctx.Done()
}

package main

import (
	"io"
	"log"
	"net"

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
			log.Printf("client finished")
			// Если клиент завершил отправку
			return nil
		}
		if err != nil {
			return err
		}

		log.Printf("Received message body from client: %s", msg.Body)

		// Отправляем сообщение обратно клиенту
		if err := stream.Send(&pb.Message{Body: msg.Body}); err != nil {
			return err
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatServer(s, &server{})
	log.Printf("Server is listening on port :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

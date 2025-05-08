package main

import (
	"context"
	"log"
	"time"

	pb "github.com/Jonnydevp/subpub/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Ошибка подключения:", err)
	}
	defer conn.Close()

	client := pb.NewPubSubClient(conn)

	// Подписка
	stream, err := client.Subscribe(context.Background(), &pb.SubscribeRequest{Key: "test"})
	if err != nil {
		log.Fatal("Ошибка подписки:", err)
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Fatal("Ошибка получения:", err)
			}
			log.Printf("Получено сообщение: %s", msg.GetData())
		}
	}()

	// Публикация
	_, err = client.Publish(context.Background(), &pb.PublishRequest{
		Key:  "test",
		Data: "Привет от клиента!",
	})
	if err != nil {
		log.Fatal("Ошибка публикации:", err)
	}

	time.Sleep(3 * time.Second) // Ждём получение
}

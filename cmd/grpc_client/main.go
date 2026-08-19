package main

import (
	"context"
	"log"

	"github.com/brianvoe/gofakeit/v6"
	ufoV1 "github.com/deelbak/grpc/pkg/proto/ufo/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverAddress = "localhost:50051"
)

func CreateUFO(ctx context.Context, client ufoV1.UfoServiceClient) (string, error) {
	name := gofakeit.LetterN(10)
	description := gofakeit.LetterN(20)

	info, err := client.Create(ctx, &ufoV1.CreateRequest{
		Name:        name,
		Description: description,
	})

	if err != nil {
		return "", err
	}
	return info.GetUfo().GetId(), nil
}

func GetUFO(ctx context.Context, client ufoV1.UfoServiceClient, id string) (*ufoV1.UFO, error) {
	info, err := client.Get(ctx, &ufoV1.GetRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}
	return info.GetUfo(), nil
}

func main() {
	ctx := context.Background()

	conn, err := grpc.NewClient(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Error creating gRPC client: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing gRPC client connection: %v", err)
		}
	}()

	client := ufoV1.NewUfoServiceClient(conn)

	log.Println("API Testing for work with UFOs")
	log.Println()

	id, err := CreateUFO(ctx, client)
	if err != nil {
		log.Printf("Error creating UFO: %v", err)
		return
	}
	log.Printf("Created UFO with ID: %s", id)

	log.Println()

	ufo, err := GetUFO(ctx, client, id)
	if err != nil {
		log.Printf("Error getting UFO: %v", err)
		return
	}
	log.Printf("Retrieved UFO: ID=%s, Name=%s, Description=%s", ufo.GetId(), ufo.GetName(), ufo.GetDescription())
}

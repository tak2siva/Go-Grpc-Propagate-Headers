package main

import (
	"fmt"
	"context"
	"grpc_server/lib/proto"
	"google.golang.org/grpc"
	"net"
	"log"
	"google.golang.org/grpc/reflection"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"time"
)

type server struct {}

func (s *server) SayHello(ctx context.Context, in *api.PingMessage) (*api.PingMessage, error) {
	log.Printf("Serving request: %s", in.Greeting)
	callServerB()
	return &api.PingMessage{Greeting: "Hello from serverA"}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 7777))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
		)),
	)
	api.RegisterPingServer(s, &server{})
	reflection.Register(s)

	log.Print("Started grpc serverA..")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}

func callServerB()  {
	conn, err := grpc.Dial("localhost:7778", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()
	c := api.NewPingClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SayHello(ctx, &api.PingMessage{Greeting:"Hello from serverA"})

	if err != nil {
		log.Fatalf("could not ping: %v", err)
	}

	log.Printf("Response from serverB %s", r.Greeting)
}
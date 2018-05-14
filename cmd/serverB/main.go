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
)

type server struct {}

func (s *server) SayHello(ctx context.Context, in *api.PingMessage) (*api.PingMessage, error) {
	log.Printf("Serving request: %s", in.Greeting)
	return &api.PingMessage{Greeting: "in serverB"}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 7778))
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

	log.Print("Started grpc serverB..")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}
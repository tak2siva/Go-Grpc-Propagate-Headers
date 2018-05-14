package main

import (
	"google.golang.org/grpc"
	"log"
	"grpc_server/lib/proto"
	"time"
	"context"
	"google.golang.org/grpc/metadata"
)

func main()  {
	conn, err := grpc.Dial("localhost:9000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()
	c := api.NewPingClient(conn)

	md, ok := metadata.FromOutgoingContext(context.Background())
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	//md = metadata.New(map[string]string{"X-Request-Id": "123khj-asdf2341-234zxc1-3452sFDaa"})

	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	r, err := c.SayHello(ctx, &api.PingMessage{Greeting:"Hello from client"})

	if err != nil {
		log.Fatalf("could not ping: %v", err)
	}

	log.Printf("Response from serverA %s", r.Greeting)
}
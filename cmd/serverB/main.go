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
	"google.golang.org/grpc/metadata"
	"strings"
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
			ZipkinUnaryInterceptorIncoming,
		)),
	)
	api.RegisterPingServer(s, &server{})
	reflection.Register(s)

	log.Print("Started grpc serverB..")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}

func ZipkinUnaryInterceptorIncoming(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}

	headers := [7]string{"X-Ot-Span-Context", "X-Request-Id", "X-B3-TraceId", "X-B3-SpanId", "X-B3-ParentSpanId", "X-B3-Sampled", "X-B3-Flags"}

	for _, header := range headers {
		headerLowerCase := strings.ToLower(header)
		if (len(md[headerLowerCase]) > 0) {
			grpc_ctxtags.Extract(ctx).Set(header, md[headerLowerCase][0])
		}
	}

	log.Printf("In Interceptor %s", grpc_ctxtags.Extract(ctx).Values())
	log.Printf("In Interceptor MD: %s", md)

	return handler(ctx, req)
}
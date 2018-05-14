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
	"google.golang.org/grpc/metadata"
	"strings"
)

type server struct {}

func (s *server) SayHello(ctx context.Context, in *api.PingMessage) (*api.PingMessage, error) {
	log.Printf("Serving request: %s", in.Greeting)

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Printf("Request Id: ", md["x-request-id"])
	}

	callServerB(ctx)
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
			ZipkinUnaryInterceptorIncoming,
		)),
	)
	api.RegisterPingServer(s, &server{})
	reflection.Register(s)

	log.Print("Started grpc serverA..")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}

func callServerB(ctx context.Context)  {
	conn, err := grpc.Dial("serverB:7778", grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc.UnaryClientInterceptor(ZipkinClientInterceptor)))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()
	c := api.NewPingClient(conn)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Printf("Request Id: ", md["x-request-id"])
	}


	r, err := c.SayHello(ctx, &api.PingMessage{Greeting:"Hello from serverA"})

	if err != nil {
		log.Fatalf("could not ping: %v", err)
	}

	log.Printf("Response from serverB %s", r.Greeting)
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

func ZipkinClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	headers := [7]string{"X-Ot-Span-Context", "X-Request-Id", "X-B3-TraceId", "X-B3-SpanId", "X-B3-ParentSpanId", "X-B3-Sampled", "X-B3-Flags"}
	outgoingContext := ctx

	for _, header := range headers {
		tags := grpc_ctxtags.Extract(ctx)
		if tags.Has(header) {
			value := tags.Values()[header]
			outgoingContext = metadata.AppendToOutgoingContext(outgoingContext, header, value.(string))
			log.Printf("Out Intercept Appending %s - %s", header, value.(string))
		}
	}

	log.Printf("Out Interceptor %s", grpc_ctxtags.Extract(ctx).Values())
	return invoker(outgoingContext, method, req, reply, cc, opts...)
}
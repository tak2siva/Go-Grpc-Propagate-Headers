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
	newCtx := context.WithValue(ctx, "user_id", "john@example.com")

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}

	grpc_ctxtags.Extract(newCtx).Set("X-Request-Id", md["x-request-id"][0])

	log.Printf("Income Interceptor %s", grpc_ctxtags.Extract(newCtx).Values())
	log.Printf("Request Id: ", md["x-request-id"])
	return handler(newCtx, req)
}

func ZipkinClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// new metadata, or copy of existing
	//incomingMD, ok := metadata.FromIncomingContext(ctx)
	//if !ok {
	//	incomingMD = metadata.New(nil)
	//}

	//has := grpc_ctxtags.Extract(ctx).Has("ThreadLocal")
	log.Printf("Out Interceptor %s", grpc_ctxtags.Extract(ctx).Values()["X-Request-Id"])

	//SPAN_CONTEXT_HEADER := "X-Ot-Span-Context"
	//REQUEST_ID_HEADER := "X-Request-Id"
	//TRACE_ID_HEADER := "X-B3-TraceId"
	//SPAN_ID_HEADER := "X-B3-SpanId"
	//PARENT_SPAN_ID_HEADER := "X-B3-ParentSpanId"
	//SAMPLED_HEADER := "X-B3-Sampled"
	//FLAGS_HEADER := "X-B3-Flags"
	//
	//
	//requestID := incomingMD[strings.ToLower(REQUEST_ID_HEADER)]
	//spanCtx := incomingMD[strings.ToLower(SPAN_CONTEXT_HEADER)]
	//traceID := incomingMD[strings.ToLower(TRACE_ID_HEADER)]
	//spanID := incomingMD[strings.ToLower(SPAN_ID_HEADER)]
	//parentSpanID := incomingMD[strings.ToLower(PARENT_SPAN_ID_HEADER)]
	//sampled := incomingMD[strings.ToLower(SAMPLED_HEADER)]
	//flag := incomingMD[strings.ToLower(FLAGS_HEADER)]
	//
	//log.Printf("%s %s %s %s %s %s %s", requestID, spanCtx, traceID, spanID, parentSpanID, sampled, flag)
	//
	////ctx = metadata.AppendToOutgoingContext(ctx, SPAN_CONTEXT_HEADER, spanCtx,)
	//
	//incomingMD, ok = metadata.FromOutgoingContext(ctx)
	//if !ok {
	//	incomingMD = metadata.New(nil)
	//} else {
	//	incomingMD = incomingMD.Copy()
	//}


	return invoker(ctx, method, req, reply, cc, opts...)
}
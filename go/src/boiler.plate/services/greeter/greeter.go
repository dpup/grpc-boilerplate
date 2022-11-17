package greeter

import (
	"context"

	"google.golang.org/grpc/grpclog"
)

func New(log grpclog.LoggerV2) GreeterServer {
	return &server{log: log}
}

// Implements GreeterServer from greeter_grpc.pb.go
type server struct {
	UnimplementedGreeterServer
	log grpclog.LoggerV2
}

func (s *server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	s.log.Infof("📧 Greeting request for %s", in.GetName())
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}

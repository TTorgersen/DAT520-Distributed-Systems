// Leave an empty line above this comment.
package main

import (
	"context"
	"fmt"
	"net"

	pb "dat520/lab3/distributedLeaderDetector/proto"

	"google.golang.org/grpc"
)

type distributedLeaderDetectorServer struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *distributedLeaderDetectorServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Listener started on %v\n", ":9000")
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGreeterServer(grpcServer, &distributedLeaderDetectorServer{})
	err = grpcServer.Serve(listener)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
)

func main(){
	lis, err := net.Listen("tcp", ":9000")
	if err != nil{
		log.Fatalf("Failed to listen on port 9000: %v", err)
	}

	grpcServer1 := grpc.NewServer()

	if err := grpcServer1.Serve(lis); err != nil{
		log.Fatalf("Failed to serve grpc server over port 9000 %v", err)
	}

}
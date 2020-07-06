package main

import (
	"fmt"
	"log"
	schedulerproto "mini-faas/scheduler/proto"
	"mini-faas/scheduler/server"
	"net"

	"google.golang.org/grpc"
)

func main() {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 7777))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	schedulerproto.RegisterSchedulerServer(s, &server.Server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

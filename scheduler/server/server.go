package server

import (
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"mini-faas/scheduler/core"
	"mini-faas/scheduler/model"
	pb "mini-faas/scheduler/proto"
)

type Server struct {
	sync.WaitGroup
	router *core.Router
}

func NewServer(router *core.Router) *Server {
	return &Server{
		router: router,
	}
}

func (s *Server) Start() {
	// Just in case the router has internal loops.
	// s.router.Start()
}

func (s *Server) AcquireContainer(ctx context.Context, req *pb.AcquireContainerRequest) (*pb.AcquireContainerReply, error) {
	if req.AccountId == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "account ID cannot be empty")
	}
	if req.FunctionConfig == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "function config cannot be nil")
	}

	reply, err := s.router.AcquireContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *Server) ReturnContainer(ctx context.Context, req *pb.ReturnContainerRequest) (*pb.ReturnContainerReply, error) {
	err := s.router.ReturnContainer(ctx, &model.ResponseInfo{
		ID:          req.RequestId,
		ContainerId: req.ContainerId,
	})
	if err != nil {
		return nil, err
	}

	return &pb.ReturnContainerReply{}, nil
}

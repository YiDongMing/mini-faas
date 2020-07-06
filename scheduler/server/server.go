package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	schedulerproto "mini-faas/scheduler/proto"
)

type Server struct {
}

func (*Server) AcquireContainer(ctx context.Context, req *schedulerproto.AcquireContainerRequest) (*schedulerproto.AcquireContainerReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AcquireContainer not implemented")
}
func (*Server) ReturnContainer(ctx context.Context, req *schedulerproto.ReturnContainerRequest) (*schedulerproto.ReturnContainerReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReturnContainer not implemented")
}

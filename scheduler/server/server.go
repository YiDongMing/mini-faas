package server

import (
	"mini-faas/scheduler/utils/logger"
	"sync"
	"time"

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
	s.router.Start()
}

var getNodecount = 5

func (s *Server) AcquireContainer(ctx context.Context, req *pb.AcquireContainerRequest) (*pb.AcquireContainerReply, error) {
	containerCC := make(chan struct{}, getNodecount)
	defer close(containerCC)
	if req.AccountId == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "account ID cannot be empty")
	}
	if req.FunctionConfig == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "function config cannot be nil")
	}

	logger.WithFields(logger.Fields{
		"Operation":     "AcquireContainer",
		"FunctionName":  req.FunctionName,
		"RequestId":     req.RequestId,
		"MemoryInBytes": req.FunctionConfig.MemoryInBytes,
	}).Infof("")
	now := time.Now().UnixNano()
	containerCC <- struct{}{}
	reply, err := s.router.AcquireContainer(ctx, req)
	<-containerCC // 执行完毕，释放资源
	if err != nil {
		logger.WithFields(logger.Fields{
			"Operation": "AcquireContainer",
			"Latency":   (time.Now().UnixNano() - now) / 1e6,
			"Error":     true,
		}).Errorf("Failed to acquire due to %v", err)
		return nil, err
	}
	return reply, nil
}

func (s *Server) ReturnContainer(ctx context.Context, req *pb.ReturnContainerRequest) (*pb.ReturnContainerReply, error) {
	logger.WithFields(logger.Fields{
		"Operation":             "ReturnContainer",
		"ContainerId":           req.ContainerId,
		"RequestId":             req.RequestId,
		"ErrorCode":             req.ErrorCode,
		"ErrorMessage":          req.ErrorMessage,
		"MaxMemoryUsageInBytes": req.MaxMemoryUsageInBytes,
		"DurationInMs":          req.DurationInNanos / 1e6,
	}).Infof("")
	now := time.Now().UnixNano()
	err := s.router.ReturnContainer(ctx, &model.ResponseInfo{
		ID:                    req.RequestId,
		ContainerId:           req.ContainerId,
		MaxMemoryUsageInBytes: req.MaxMemoryUsageInBytes,
		DurationInMs:          req.DurationInNanos,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"Operation": "ReturnContainer",
			"Latency":   (time.Now().UnixNano() - now) / 1e6,
			"Error":     true,
		}).Errorf("Failed to acquire due to %v", err)
		return nil, err
	}

	return &pb.ReturnContainerReply{}, nil
}

package core

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	pb "mini-faas/nodeservice/proto"
)

type NodeInfo struct {
	sync.Mutex
	nodeID              string
	address             string
	port                int64
	availableMemInBytes int64

	conn *grpc.ClientConn
	pb.NodeServiceClient
}

func NewNode(nodeID, address string, port, memory int64) (*NodeInfo, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", address, port), grpc.WithInsecure())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &NodeInfo{
		Mutex:               sync.Mutex{},
		nodeID:              nodeID,
		address:             address,
		port:                port,
		availableMemInBytes: memory,
		conn:                conn,
		NodeServiceClient:   pb.NewNodeServiceClient(conn),
	}, nil
}

// Close closes the connection
func (n *NodeInfo) Close() {
	n.conn.Close()
}

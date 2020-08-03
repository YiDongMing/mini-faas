package core

import (
	"context"
	"fmt"
	"mini-faas/scheduler/utils/logger"
	"sort"
	"strconv"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	nsPb "mini-faas/nodeservice/proto"
	rmPb "mini-faas/resourcemanager/proto"
	cp "mini-faas/scheduler/config"
	"mini-faas/scheduler/model"
	pb "mini-faas/scheduler/proto"
)

type ContainerInfo struct {
	sync.Mutex
	id       string
	address  string
	port     int64
	nodeId   string
	requests map[string]int64 // request_id -> status
}

type Router struct {
	nodeMap           cmap.ConcurrentMap // instance_id -> NodeInfo
	functionMap       cmap.ConcurrentMap // function_name -> ContainerMap (container_id -> ContainerInfo)
	requestMap        cmap.ConcurrentMap // request_id -> FunctionName
	rmClient          rmPb.ResourceManagerClient
	nodeUpdateTimeMap cmap.ConcurrentMap //last use node time
	functionInfoMap   cmap.ConcurrentMap //record the use memery of function
}

func NewRouter(config *cp.Config, rmClient rmPb.ResourceManagerClient) *Router {
	return &Router{
		nodeMap:           cmap.New(),
		functionMap:       cmap.New(),
		requestMap:        cmap.New(),
		rmClient:          rmClient,
		nodeUpdateTimeMap: cmap.New(),
		functionInfoMap:   cmap.New(),
	}
}

func (r *Router) Start() {
	// Just in case the router has internal loops.
	//程序开始时先申请5个node，防止最开始的请求因为获取不到node而失败
	for i := 0; i < 5; i++ {
		r.createNewNode()
	}
}

func (r *Router) createNewNode() {
	ctxR, cancelR := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelR()
	now := time.Now().UnixNano()
	var replyRn = new(rmPb.ReserveNodeReply)
	var err error
	replyRn, err = r.rmClient.ReserveNode(ctxR, &rmPb.ReserveNodeRequest{
		AccountId: "1",
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"Operation": "ReserveNode",
			"Latency":   (time.Now().UnixNano() - now) / 1e6,
			"Error":     true,
		}).Errorf("Failed to reserve node due to %v", err)
		return
	}
	fmt.Println(replyRn)
	var nodeDesc = new(rmPb.NodeDesc)
	nodeDesc = replyRn.Node
	node, err := NewNode(nodeDesc.Id, nodeDesc.Address, nodeDesc.NodeServicePort, nodeDesc.MemoryInBytes)
	if err != nil {
		r.rmClient.ReleaseNode(ctxR, &rmPb.ReleaseNodeRequest{Id: nodeDesc.GetId()})
	}
	r.nodeMap.Set(nodeDesc.Id, node)
}

func (r *Router) addNodeAuto() {
	var allNodeNum int = 20
	var weight int = 5
	fmt.Print("begin to check node")
	nodeMap := r.nodeMap
	if nodeMap.Count() < allNodeNum {
		//当node使用数小于最大值20时，则按照weight来决定从可以申请的node数中决定要提前申请node的数量
		//假设node为5。而weight为0.5。node不超过10则按照node的使用数乘以weight 则需要申请的数量为5*0.5=2.5约等于2，
		//假设node为12，而weight为0.5。node能申请的数量小于10，则将剩余的数量乘以weight，则需要提前申请的数量为8*0.5等于4
		if nodeMap.Count() < allNodeNum/2 {
			var size = (nodeMap.Count() * weight) / 10
			for i := 0; i < size; i++ {
				r.createNewNode()
				logger.WithFields(logger.Fields{
					"Operation": "createNewNode1",
					"count":     i,
				}).Infof("begin to createNewNode1-")
			}
		} else {
			var size = ((allNodeNum - nodeMap.Count()) * weight) / 10
			for i := 0; i < size; i++ {
				r.createNewNode()
				logger.WithFields(logger.Fields{
					"Operation": "createNewNode2",
					"count":     i,
				}).Infof("begin to createNewNode2-")
			}
		}
	}
}
func (r *Router) ReduceNodeAuto() {
	for true {
		time.Sleep(30 * time.Second)
		var nodeId = ""
		for _, key := range sortedKeys(r.nodeMap.Keys()) {
			nmObj, _ := r.nodeMap.Get(key)
			node := nmObj.(*NodeInfo)
			node.Lock()
			ctxR, cancelR := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelR()
			nodeStats, _ := node.GetStats(ctxR, &nsPb.GetStatsRequest{RequestId: node.nodeID})
			if nodeStats.GetContainerStatsList() == nil {
				nodeId = key
				node.Unlock()
				break
			}
			node.Unlock()
		}
		if nodeId != "" {
			nmObj, _ := r.nodeMap.Get(nodeId)
			node := nmObj.(*NodeInfo)
			node.Lock()
			ctxR, cancelR := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelR()
			nodeStats, _ := node.GetStats(ctxR, &nsPb.GetStatsRequest{RequestId: node.nodeID})
			if nodeStats.GetContainerStatsList() == nil {
				r.nodeMap.Remove(nodeId)
				_, err := r.rmClient.ReleaseNode(ctxR, &rmPb.ReleaseNodeRequest{Id: node.nodeID})
				if err != nil {
					logger.WithFields(logger.Fields{
						"Operation": "MyReleaseNode",
						"Error":     true,
					}).Errorf("Failed to release node %v", err)
				}
				r.nodeMap.Remove(nodeId)
				logger.WithFields(logger.Fields{
					"Operation":           "ReleaseNode",
					"nodeId":              node.nodeID,
					"availableMemInBytes": node.availableMemInBytes,
					"GetLiveId":           nodeStats.GetLiveId(),
					"GetNodeStats":        nodeStats.GetNodeStats(),
				}).Infof("sucess to release node")
				node.Unlock()
			}

		}
	}

}

func (r *Router) AcquireContainer(ctx context.Context, req *pb.AcquireContainerRequest) (*pb.AcquireContainerReply, error) {
	var res *ContainerInfo
	fmt.Print("requestNum:::::" + strconv.Itoa(r.requestMap.Count()))
	logger.WithFields(logger.Fields{
		"Operation": "requestNum",
		"count":     r.requestMap.Count(),
	}).Infof("")
	fmt.Print("functionNum:::::" + strconv.Itoa(r.functionMap.Count()))
	fmt.Print("nodeNum:::::" + strconv.Itoa(r.nodeMap.Count()))

	// Save the name for later ReturnContainer
	r.requestMap.Set(req.RequestId, req.FunctionName)

	r.functionMap.SetIfAbsent(req.FunctionName, cmap.New())
	fmObj, _ := r.functionMap.Get(req.FunctionName)
	containerMap := fmObj.(cmap.ConcurrentMap)
	funMemory, ok := r.functionInfoMap.Get(req.FunctionName)
	if ok {
		memory := funMemory.(int64)
		if (req.FunctionConfig.MemoryInBytes / memory) > 2 {
			for _, key := range sortedKeys(containerMap.Keys()) {
				cmObj, _ := containerMap.Get(key)
				container := cmObj.(*ContainerInfo)
				container.Lock()
				if container.requests[req.RequestId] < (req.FunctionConfig.MemoryInBytes / memory) {
					container.requests[req.RequestId]++
					res = container
					container.Unlock()
					break
				} else if len(container.requests) < 1 {
					container.requests[req.RequestId] = 1
					res = container
					container.Unlock()
					break
				}
				container.Unlock()
			}
		}
	}
	for _, key := range sortedKeys(containerMap.Keys()) {
		cmObj, _ := containerMap.Get(key)
		container := cmObj.(*ContainerInfo)
		container.Lock()
		if len(container.requests) < 1 {
			container.requests[req.RequestId] = 1
			res = container
			container.Unlock()
			break
		}
		container.Unlock()
	}

	if res == nil { // if no idle container exists
		node, err := r.getNode(req.AccountId, req.FunctionConfig.MemoryInBytes)
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		replyC, err := node.CreateContainer(ctx, &nsPb.CreateContainerRequest{
			Name: req.FunctionName + uuid.NewV4().String(),
			FunctionMeta: &nsPb.FunctionMeta{
				FunctionName:  req.FunctionName,
				Handler:       req.FunctionConfig.Handler,
				TimeoutInMs:   req.FunctionConfig.TimeoutInMs,
				MemoryInBytes: req.FunctionConfig.MemoryInBytes,
			},
			RequestId: req.RequestId,
		})
		if err != nil {
			r.handleContainerErr(node, req.FunctionConfig.MemoryInBytes)
			return nil, errors.Wrapf(err, "failed to create container on %s", node.address)
		}
		res = &ContainerInfo{
			id:       replyC.ContainerId,
			address:  node.address,
			port:     node.port,
			nodeId:   node.nodeID,
			requests: make(map[string]int64),
		}
		res.requests[req.RequestId] = 1 // The container hasn't been listed in the containerMap. So we don't need locking here.
		containerMap.Set(res.id, res)
	}

	return &pb.AcquireContainerReply{
		NodeId:          res.nodeId,
		NodeAddress:     res.address,
		NodeServicePort: res.port,
		ContainerId:     res.id,
	}, nil
}

func (r *Router) getNode(accountId string, memoryReq int64) (*NodeInfo, error) {
	useNodeFlag := 0
	for _, key := range sortedKeys(r.nodeMap.Keys()) {
		nmObj, _ := r.nodeMap.Get(key)
		node := nmObj.(*NodeInfo)
		node.Lock()
		useNodeFlag++
		if node.availableMemInBytes > memoryReq {
			node.availableMemInBytes -= memoryReq
			node.Unlock()
			if useNodeFlag == r.nodeMap.Count() {
				go r.addNodeAuto()
			}
			return node, nil
		}
		node.Unlock()
	}
	ctxR, cancelR := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelR()
	now := time.Now().UnixNano()
	replyRn, err := r.rmClient.ReserveNode(ctxR, &rmPb.ReserveNodeRequest{
		AccountId: accountId,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"Operation": "ReserveNode",
			"Latency":   (time.Now().UnixNano() - now) / 1e6,
			"Error":     true,
		}).Errorf("Failed to reserve node due to %v", err)
		return nil, errors.WithStack(err)
	}
	logger.WithFields(logger.Fields{
		"Operation": "ReserveNode",
		"Latency":   (time.Now().UnixNano() - now) / 1e6,
		"accountId": accountId,
	}).Infof("")

	nodeDesc := replyRn.Node
	node, err := NewNode(nodeDesc.Id, nodeDesc.Address, nodeDesc.NodeServicePort, nodeDesc.MemoryInBytes+536870912) //加上512MB，只留512MB给node
	if err != nil {
		// : Release the Node
		r.rmClient.ReleaseNode(ctxR, &rmPb.ReleaseNodeRequest{Id: nodeDesc.GetId()})
		return nil, err
	}
	r.nodeMap.Set(nodeDesc.Id, node)
	return node, nil
}

func (r *Router) handleContainerErr(node *NodeInfo, functionMem int64) {
	node.Lock()
	node.availableMemInBytes += functionMem
	node.Unlock()
}

func (r *Router) ReturnContainer(ctx context.Context, res *model.ResponseInfo) error {
	rmObj, ok := r.requestMap.Get(res.ID)
	if !ok {
		return errors.Errorf("no request found with id %s", res.ID)
	}
	fmObj, ok := r.functionMap.Get(rmObj.(string))
	if !ok {
		return errors.Errorf("no container acquired for the request %s", res.ID)
	}
	containerMap := fmObj.(cmap.ConcurrentMap)
	cmObj, ok := containerMap.Get(res.ContainerId)
	if !ok {
		return errors.Errorf("no container found with id %s", res.ContainerId)
	}
	container := cmObj.(*ContainerInfo)
	nodeId := container.nodeId
	nmObj, _ := r.nodeMap.Get(nodeId)
	if nmObj != nil {
		container.Lock()
		if container.requests[res.ID] == 1 {
			delete(container.requests, res.ID)
		} else {
			container.requests[res.ID]--
		}
		container.Unlock()
		fnob, _ := r.requestMap.Get(res.ID)
		functionName := fnob.(string)
		r.functionInfoMap.Set(functionName, res.MaxMemoryUsageInBytes)
		r.requestMap.Remove(res.ID)
		/*_, err := node.RemoveContainer(ctxR, &nsPb.RemoveContainerRequest{ContainerId: res.ContainerId})
		if err != nil{
			logger.WithFields(logger.Fields{
				"Operation": "RemoveContainer",
				"ContainerId":   res.ContainerId,
				"nodeId": nodeId,
			}).Infof("")
			container.Unlock()
			return err
		}else{

			return nil
		}*/
		return nil
	}
	return nil
}

func sortedKeys(keys []string) []string {
	sort.Strings(keys)
	return keys
}

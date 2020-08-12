package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/spf13/viper"
	"google.golang.org/grpc"

	rmPb "mini-faas/resourcemanager/proto"
	cp "mini-faas/scheduler/config"
	"mini-faas/scheduler/core"
	pb "mini-faas/scheduler/proto"
	"mini-faas/scheduler/server"
	"mini-faas/scheduler/utils/env"
	"mini-faas/scheduler/utils/global"
	"mini-faas/scheduler/utils/logger"
)

const (
	defaultConfigFile    = "config/dev/config.json"
	defaultLogConfigFile = "config/dev/log.xml"
	defaultPort          = 10450
	moduleName           = "Scheduler"
)

func main() {
	var configFile = flag.String(
		"config", defaultConfigFile, "path to the configuration file")
	var logConfigFile = flag.String(
		"logconfig", defaultLogConfigFile, "path to the log configuration")

	flag.Parse()

	env.InitLogger("Logger_Default", *logConfigFile)

	defer logger.Flush()
	defer logger.Infof("Scheduler gRPC server exited.")

	hostName, err := env.GetHostName()
	if err != nil {
		logger.Errorf("Failed to get host name due to %s", err)
		return
	}

	global.HostName = hostName
	global.ModuleName = moduleName
	cp.Global.HostName = hostName

	err = env.ViperConfig(global.ModuleName, *configFile)
	if err != nil {
		logger.Criticalf("Failed to viper config for Scheduler in config file %s due to %s", *configFile, err)
		panic(err)
	}

	var config cp.Config
	err = viper.Unmarshal(&config)
	if err != nil {
		logger.Criticalf("Failed to decode viper config, %v", err)
		panic(err)
	}
	cp.Global = &config

	done := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())

	go env.HandleSignal(cancel, done)
	svr := grpc.NewServer()

	rmEndpoint := os.Getenv("RESOURCE_MANAGER_ENDPOINT")
	if rmEndpoint == "" {
		panic("environment variable RESOURCE_MANAGER_ENDPOINT is not set")
	}
	logger.Infof("Creating resource manager client with endpoint %s", rmEndpoint)

	conn, err := grpc.Dial(rmEndpoint, grpc.WithInsecure())
	if err != nil {
		logger.Criticalf("Failed to contact resource manager due to %s", err)
		panic(err)
	}
	rm := rmPb.NewResourceManagerClient(conn)

	router := core.NewRouter(&config, rm)
	router.Start()
	//go router.ReduceNodeAuto()
	//go router.ReduceContainer()
	s := server.NewServer(router)
	pb.RegisterSchedulerServer(svr, s)
	s.Start()
	servicePort := defaultPort
	servicePortS := os.Getenv("SERVICE_PORT")
	if servicePortS != "" {
		port, err := strconv.Atoi(servicePortS)
		if err != nil {
			panic(err)
		}
		servicePort = port
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", servicePort))
	if err != nil {
		logger.Criticalf("Failed to listen on port %d: %v", servicePort, err)
		return
	}

	go svr.Serve(lis)

	select {
	case <-ctx.Done():
		logger.Infof("Scheduler gRPC server gracefully stopping ...")
		svr.GracefulStop()
		logger.Infof("Scheduler gRPC server gracefully stopped.")
	}
}

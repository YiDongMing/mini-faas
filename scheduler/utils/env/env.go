package env

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/viper"

	"aliyun/serverless/mini-faas/scheduler/utils/logger"
	"aliyun/serverless/mini-faas/scheduler/utils/logger/seelog"
)

// getHostName : get the hostname of the host machine if the container is started by docker run --net=host
func getHostName() (string, error) {
	cmd := exec.Command("/bin/hostname")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	hostname := strings.TrimSpace(string(out))
	if hostname == "" {
		return "", fmt.Errorf("no hostname get from cmd '/bin/hostname' in the container, please check")
	}
	return hostname, nil
}

// GetHostName : get hostname of host machine
func GetHostName() (string, error) {
	hostName := os.Getenv("HOST_NAME")
	if hostName != "" {
		return hostName, nil
	}
	logger.Warningf("get HOST_NAME from env failed, is env.(\"HOST_NAME\") already set? Will use hostname instead")
	return getHostName()
}

// HandleSignal ...
func HandleSignal(cancel context.CancelFunc, done chan struct{}) {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM)
	select {
	case <-sc:
		logger.Infof("Server is killed by SIGTERM")
		cancel()
		close(done)
		logger.Flush()
	}
}

// ViperConfig ...
// Print the information in the stdout as well for debugging purpose.
func ViperConfig(moduleName, configPath string) error {
	msg := fmt.Sprintf("%s is starting", moduleName)
	fmt.Println(msg)
	logger.Infof(msg)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	// Set these keys explicitly, otherwise viper only overrides keys that are defined in the config file.
	viper.SetEnvPrefix(moduleName)
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		logger.Errorf("Failed to read config file %s due to %s", configPath, err)
		return err
	}

	logger.Infof("Dump viper configs")
	for _, k := range viper.AllKeys() {
		logger.Infof("%s: %v", k, viper.Get(k))
	}
	logger.Flush()
	return nil
}

// InitLogger ...
func InitLogger(loggerName string, configFileName string) {
	log, err := seelog.NewSeeLoggerFromFile(configFileName)
	if err != nil {
		panic(err)
	}
	if loggerName == "" {
		logger.SetLogger("Logger_Default", log)
	} else {
		logger.SetLogger(loggerName, log)
	}
}

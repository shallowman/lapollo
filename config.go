package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const defaultLocalhostIP = "127.0.0.1"

type app struct {
	Path      string   `yaml:"path"`
	AppId     string   `yaml:"appId"`
	Namespace []string `yaml:"namespace"`
}

type config struct {
	Cluster string `yaml:"cluster"`
	Type    int    `yaml:"type"`
	Host    string `yaml:"host"`
	IP      string `yaml:"ip"`
	LogPath string `yaml:"logPath"`
	Apps    []app  `yaml:"apps"`
}


func getHostIp() string {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, netInterface := range netInterfaces {
		// interface down
		if netInterface.Flags&net.FlagUp == 0 {
			continue
		}
		// loopback interface
		if netInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := netInterface.Addrs()
		if err != nil {
			return ""
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			// not an ipv4 address
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func init() {
	conf = getConfigViaOsEnvironment()
	if conf == nil {
		conf = getConfigViaYaml()
	}

	if conf == nil {
		panic("Apollo-Client 启动失败，没有对应的配置文件")
	}
}

func getConfigViaOsEnvironment() *config {
	cluster := os.Getenv("APOLLO_CLUSTER")
	apolloHost := os.Getenv("APOLLO_HOST")
	envPath := os.Getenv("APOLLO_ENV_PATH")
	appId := os.Getenv("APOLLO_APP_ID")
	namespace := os.Getenv("APOLLO_NAMESPACE")
	logPath := os.Getenv("APOLLO_CLIENT_LOG_PATH")

	if cluster == "" || apolloHost == "" || envPath == "" || appId == "" || namespace == "" {
		return nil
	}

	ip := getHostIp()

	if ip == "" {
		ip = defaultLocalhostIP
	}

	namespaces := strings.Split(namespace, ",")

	return &config{
		Cluster: cluster,
		Type:    1,
		Host:    apolloHost,
		IP:      ip,
		LogPath: logPath,
		Apps: []app{{
			envPath,
			appId,
			namespaces,
		}},
	}
}

func getConfigViaYaml() *config {
	var configYaml string

	if homePath := os.Getenv("HOME"); homePath != "" {
		configYaml = homePath + "/.lapollo/app.yaml"
	}

	if _, err := os.Stat(configYaml); err != nil {
		currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			logger.Errorf("Apollo-Client 获取路径错误, %s \n", err)
		}
		configYaml = strings.TrimRight(currentDir, "/") + "/app.yaml"
	}

	if configYaml == "" {
		logger.Error("Apollo-Client 配置文件不存在")
		return nil
	}

	contents, err := ioutil.ReadFile(configYaml)

	if err != nil {
		logger.Errorf("Apollo-Client 从 %s 读取配置时发生错误， %s \n", configYaml, err)
	}
	var config *config
	if err := yaml.Unmarshal(contents, &config); err != nil {
		logger.Errorf("Apollo-Client 解析 yaml 配置文件 %s 发生错误 %s \n", configYaml, err)
		return nil
	}
	return config
}

package client

import (
	"net"
	"os"
	"strings"
)

type App struct {
	Path      string
	AppId     string
	Namespace []string
}

type ApolloClientConfig struct {
	Cluster string
	Type    int
	Host    string
	IP      string
	Apps    []App
}

var Conf *ApolloClientConfig

func getClientIp() string {
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
	// Get config from OS environment
	cluster := os.Getenv("APOLLO_CLUSTER")
	apolloHost := os.Getenv("APOLLO_HOST")
	envPath := os.Getenv("APOLLO_ENV_PATH")
	appId := os.Getenv("APOLLO_APP_ID")
	namespace := os.Getenv("APOLLO_NAMESPACE")
	ip := getClientIp()
	if ip == "" {
		ip = "127.0.0.1"
	}
	if envPath == "" {
		envPath = "/var/www/.env"
	}
	namespaces := strings.Split(namespace, ",")
	Conf = &ApolloClientConfig{
		Cluster: cluster,
		Type:    1,
		Host:    apolloHost,
		IP:      ip,
		Apps: []App{{
			envPath,
			appId,
			namespaces,
		}},
	}
}

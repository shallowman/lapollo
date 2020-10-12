package client

import (
	"net"
	"os"
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

func getHostIp() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
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
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String()
		}
	}
	return ""
}

func init() {
	var cluster = os.Getenv("APOLLO_CLUSTER")
	var apolloHost = os.Getenv("APOLLO_HOST")
	var envPath = "/var/www/.env"
	var appId = os.Getenv("APOLLO_APP_ID")
	var namespace = os.Getenv("APOLLO_NAMESPACE")

	Conf.Cluster = cluster
	Conf.Type = 1
	Conf.Host = apolloHost

	ip := getHostIp()
	if ip == "" {
		ip = "127.0.0.1"
	}

	Conf.IP = ip

	Conf.Apps = []App{{
		envPath,
		appId,
		[]string{namespace},
	}}
}

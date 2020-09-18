package client

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var appEnv = os.Getenv("APOLLO_ENV")

type ApolloClientConfig struct {
	Cluster string `yaml:"cluster"`
	Type    int    `yaml:"type"`
	Host    string `yaml:"host"`
	IP      string `yaml:"ip"`
	Apps    []struct {
		Path      string   `yaml:"path"`
		AppId     string   `yaml:"appId"`
		Namespace []string `yaml:"namespace"`
	} `yaml:"apps"`
}

var Conf *ApolloClientConfig
var ConfigFile string

func init() {
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(fmt.Errorf("获取文件位置错误 %s \n", err))
	}

	ConfigFile = strings.TrimRight(currentDir, "/") + "/app.yaml"
	contents, err := ioutil.ReadFile(ConfigFile)

	if err != nil {
		panic(fmt.Errorf("读取 app.yaml 文件内容错误 : %s \n", err))
	}

	err = yaml.Unmarshal(contents, &Conf)
	if err != nil {
		panic(fmt.Errorf("yaml 配置文件解析错误 : %s \n", err))
	}
}

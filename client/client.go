package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type HttpReqConfig struct {
	Path          string
	AppId         string
	Namespace     string
	ReleaseKey    string
	Notifications string
}

func generateViaCacheHttpReqUri(host string, appId string, cluster string, namespace string) string {
	return fmt.Sprintf("%s/configfiles/json/%s/%s/%s", host, appId, cluster, namespace)
}

func buildHttpQuery(quires map[string]string) string {
	var httpQueries []string
	for k, v := range quires {
		httpQueries = append(httpQueries, k+"="+v)
	}
	query, _ := url.ParseQuery(strings.Join(httpQueries, "&"))
	return query.Encode()
}

func generateViaNotificationsHttpReqUri(host string, query string) string {
	return fmt.Sprintf("%s/notifications/v2?%s", host, query)
}

func generateNotViaCacheHttpReqUri(host string, appId string, cluster string, namespace string, releaseKey string) string {
	return fmt.Sprintf("%s/configs/%s/%s/%s?releaseKey=%s", host, appId, cluster, namespace, releaseKey)
}

//通过带缓存的接口从 Apollo Server 读取配置
func getConfigWithCache(config HttpReqConfig) (responseBody map[string]string) {
	requestUri := generateViaCacheHttpReqUri(Conf.Host, config.AppId, Conf.Cluster, config.Namespace)

	if Conf.IP != "" {
		requestUri += "?ip=" + Conf.IP
	}

	response, err := http.Get(requestUri)

	if err != nil {
		Logger.Fatal("[带缓存接口获取配置] apollo client 发送 HTTP 请求错误: " + err.Error())
	}

	if response == nil {
		Logger.Error("[带缓存接口获取配置] apollo-server http 应答为空")
		return
	}

	if response.StatusCode == 200 {
		if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
			Logger.Fatalf("[带缓存接口获取配置] 反序列化 JSON 串错误: %v %v", err.Error(), response.Body)
		}
	} else {
		Logger.Errorf("[带缓存接口获取配置]apollo-server HTTP 响应码非 200 %v %v\n", response.StatusCode, response.Body)
	}

	return
}

//应用感知配置更新
func getNotifications(config HttpReqConfig) (bool, int64) {
	query := map[string]string{
		"appId":         config.AppId,
		"cluster":       Conf.Cluster,
		"notifications": config.Notifications,
	}

	notificationsUri := generateViaNotificationsHttpReqUri(Conf.Host, buildHttpQuery(query))
	response, err := http.Get(notificationsUri)

	if err != nil {
		Logger.Fatal("[长轮询接口获取配置] GoHttpClient 错误" + err.Error())
	}

	if response == nil {
		Logger.Fatal("[长轮询接口获取配置] apollo server 接口返回为空")
		return false, 0
	}

	defer response.Body.Close()

	var body []struct {
		Namespace      string `json:"namespaceName"`
		NotificationId int64  `json:"notificationId"`
		Messages       struct {
			Details map[string]int64 `json:"details"`
		} `json:"messages"`
	}
	if response.StatusCode == 200 {
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			Logger.Error("[长轮询接口获取配置] JSON 反序列化返回的消息体发生错误: %v %v", err.Error(), response.Body)
		}
		return true, body[0].NotificationId
	}
	return false, 0
}

//通过不带缓存的Http接口从Apollo读取配置
func getConfigWithoutCache(config HttpReqConfig) (string, map[string]string) {
	requestUri := generateNotViaCacheHttpReqUri(Conf.Host, config.AppId, Conf.Cluster, config.Namespace, config.ReleaseKey)
	if Conf.IP != "" {
		requestUri += "&ip=" + Conf.IP
	}

	response, err := http.Get(requestUri)

	if err != nil {
		Logger.Fatal("[不带缓存接口获取配置] GoHttpClient 错误" + err.Error())
		return "", map[string]string{}
	}

	var body struct {
		AppId          string            `json:"appId"`
		Cluster        string            `json:"cluster"`
		Namespace      string            `json:"Namespace"`
		Configurations map[string]string `json:"configurations"`
		ReleaseKey     string            `json:"releaseKey"`
	}

	if response.StatusCode == 200 {
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			Logger.Fatal("[不带缓存接口获取配置] JSON 反序列化Http接口返回的消息体发生错误： %v %v", err.Error(), response.Body)
		}
		return body.ReleaseKey, body.Configurations
	}
	Logger.Fatalf("[不带缓存接口获取配置] Http 接口返回码非 200 %v %v\n", response.StatusCode, response.Body)
	return "", map[string]string{}
}

func updateEnvUnderNamespace(path string, namespace string, configs map[string]string) {
	envContents := ""
	for k, v := range configs {
		envContents += k + "=" + v + "\n"
	}
	// 写入新 env 前会清空之前的 env
	err := ioutil.WriteFile(path+"/apollo.config."+namespace, []byte(envContents), 0777)
	if err != nil {
		Logger.Fatal(err)
	}
}

// 轮询更新
func UpdateViaHttpPolling(config HttpReqConfig, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			configs := getConfigWithCache(config)
			updateEnvUnderNamespace(filepath.Dir(config.Path), config.Namespace, configs)
			time.Sleep(30 * time.Second)
		}
	}
}

// 长轮询更新，热更新
func UpdateEnvViaHttpLongPolling(config HttpReqConfig, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	var notificationId int64
	var releaseKey string
	for {
		select {
		case <-ctx.Done():
			return
		default:
			notifications, _ := json.Marshal([]map[string]interface{}{{
				"namespaceName":  config.Namespace,
				"notificationId": notificationId,
			}})
			config.Notifications = string(notifications)
			updated, id := getNotifications(config)
			if updated {
				var configs map[string]string
				config.ReleaseKey = releaseKey
				releaseKey, configs = getConfigWithoutCache(config)
				updateEnvUnderNamespace(filepath.Dir(config.Path), config.Namespace, configs)
				notificationId = id
			}
		}
	}
}

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

func getHttpCacheUri(host string, appId string, cluster string, namespace string) string {
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

func getNotificationsUri(host string, query string) string {
	return fmt.Sprintf("%s/notifications/v2?%s", host, query)
}

func getHttpWithoutCacheUri(host string, appId string, cluster string, namespace string, releaseKey string) string {
	return fmt.Sprintf("%s/configs/%s/%s/%s?releaseKey=%s", host, appId, cluster, namespace, releaseKey)
}

//通过带缓存的接口从 Apollo Server 读取配置
func getConfigWithCache(config HttpReqConfig) (responseBody map[string]string) {
	requestUri := getHttpCacheUri(Conf.Host, config.AppId, Conf.Cluster, config.Namespace)

	if Conf.IP != "" {
		requestUri += "?ip=" + Conf.IP
	}

	response, err := http.Get(requestUri)

	if err != nil {
		Logger.Fatal("通过带缓存的Http接口从Apollo读取配置时，GoHttpClient 错误" + err.Error())
	}

	if response.StatusCode == 200 {
		err = json.NewDecoder(response.Body).Decode(&responseBody)
		if err != nil {
			Logger.Fatal("通过带缓存的Http接口从Apollo读取配置时， JSON 反序列化Http接口返回的消息体发生错误" + err.Error())
		}
		return
	}

	log.Fatalf("通过带缓存的Http接口从Apollo读取配置时，客户端返回码非 200 %v %v\n", response.StatusCode, response.Body)
	return
}

//应用感知配置更新
func getNotifications(config HttpReqConfig) (bool, int64) {
	query := map[string]string{
		"appId":         config.AppId,
		"cluster":       Conf.Cluster,
		"notifications": config.Notifications,
	}

	notificationsUri := getNotificationsUri(Conf.Host, buildHttpQuery(query))
	response, err := http.Get(notificationsUri)

	if err != nil {
		Logger.Fatal("通过应用感知配置更新接口从Apollo读取配置时，GoHttpClient 错误" + err.Error())
	}

	if response == nil {
		Logger.Fatal("通过应用感知配置更新接口从Apollo读取配置时，apollo server 接口返回为空")
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
		err = json.NewDecoder(response.Body).Decode(&body)
		Logger.Info("http 返回", body)
		if err != nil {
			Logger.Error("通过应用感知配置更新接口从Apollo读取配置时，JSON 反序列化Http接口返回的消息体发生错误" + err.Error())
		}
		return true, body[0].NotificationId
	}
	return false, 0
}

//通过不带缓存的Http接口从Apollo读取配置
func getConfigWithoutCache(config HttpReqConfig) (string, map[string]string) {
	requestUri := getHttpWithoutCacheUri(Conf.Host, config.AppId, Conf.Cluster, config.Namespace, config.ReleaseKey)
	if Conf.IP != "" {
		requestUri += "&ip=" + Conf.IP
	}

	response, err := http.Get(requestUri)

	if err != nil {
		Logger.Fatal("通过不带缓存的Http接口从Apollo读取配置时，GoHttpClient 错误" + err.Error())
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
		err = json.NewDecoder(response.Body).Decode(&body)
		if err != nil {
			Logger.Fatal("通过不带缓存的Http接口从Apollo读取配置时，JSON 反序列化Http接口返回的消息体发生错误" + err.Error())
		}
		return body.ReleaseKey, body.Configurations
	}
	Logger.Fatalf("通过不带缓存的Http接口从Apollo读取配置时，Http 接口返回码非 200 %v\n", response.StatusCode)
	return "", map[string]string{}
}

func updateEnvWithNamespace(path string, namespace string, configs map[string]string) {
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
func PollingUpdate(config HttpReqConfig, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			configs := getConfigWithCache(config)
			updateEnvWithNamespace(filepath.Dir(config.Path), config.Namespace, configs)
			time.Sleep(30 * time.Second)
		}
	}
}

// 长轮询更新，热更新
func LongPollingHotUpdate(config HttpReqConfig, wg *sync.WaitGroup, ctx context.Context) {
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
				updateEnvWithNamespace(filepath.Dir(config.Path), config.Namespace, configs)
				notificationId = id
			}
		}
	}
}

package main

import (
	"context"
	"fmt"
	"github.com/lapollo/client"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
)

var (
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
)

const (
	POLLING = iota
	HOT
)

func main() {
	client.Logger.Info("start client")
	ctx, cancel = context.WithCancel(context.Background())
	updateAppsConfig(ctx)
	go handleSignal()
	wg.Wait()
	client.Logger.Info("apollo 客户端退出")
}

func updateAppsConfig(ctx context.Context) {
	for _, app := range client.Conf.Apps {
		for _, namespace := range app.Namespace {
			wg.Add(1)
			go func(path, appId, namespace string, ctx context.Context) {
				switch client.Conf.Type {
				case POLLING:
					client.PollingUpdate(client.HttpReqConfig{
						Path:      path,
						AppId:     appId,
						Namespace: namespace,
					}, &wg, ctx)
					break
				case HOT:
					client.LongPollingHotUpdate(client.HttpReqConfig{
						Path:          path,
						AppId:         appId,
						Namespace:     namespace,
						ReleaseKey:    "",
						Notifications: "",
					}, &wg, ctx)
					break
				default:
					client.Logger.Error("配置文件 type 类型错误")
					panic(fmt.Errorf("配置文件 type 类型错误 %v", client.Conf.Type))
				}
				return

			}(app.Path, app.AppId, namespace, ctx)
		}

		client.UpdateAppEnvironment(filepath.Dir(app.Path), app.Namespace)
	}
}

func handleSignal() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	for {
		switch <-signals {
		case syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT:
			client.Logger.Error("apollo 客户端正在停止")
			cancel()
		}
	}
}

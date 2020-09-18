package main

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/lapollo/client"
	"os"
	"os/signal"
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

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		client.Logger.Error("添加文件 watcher 失败", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	updateAppsConfig(ctx)

	go listenClientConfig(watcher)
	go handleSignal()

	err = watcher.Add(client.ConfigFile)

	if err != nil {
		client.Logger.Fatal(err)
	}

	wg.Wait()
	defer watcher.Close()
	client.Logger.Info("apollo 客户端退出")
}

func listenClientConfig(watcher *fsnotify.Watcher) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if (event.Op&fsnotify.Chmod == fsnotify.Chmod) || (event.Op&fsnotify.Write == fsnotify.Write) {
				client.Logger.Info("Lapollo 客服端配置更新", event)
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				updateAppsConfig(ctx)
			}
		case err, ok := <-watcher.Errors:
			client.Logger.Error("文件 watcher 过程中发生错误", err)
			if !ok {
				return
			}
		}
	}
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

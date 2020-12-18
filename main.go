package main

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"os"
	"os/exec"
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
	Logger.Info("apollo 客户端启动")
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		Logger.Error("添加文件 watcher 失败", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	continuousUpdate(ctx, watcher)
	go handleSignal()

	wg.Wait()
	defer watcher.Close()
	Logger.Info("apollo 客户端退出")
}

func continuousUpdate(ctx context.Context, watcher *fsnotify.Watcher) {
	for _, app := range Conf.Apps {
		go listenAppNamespaceConfig(watcher, filepath.Dir(app.Path), app.Namespace)
		for _, namespace := range app.Namespace {
			err1 := ioutil.WriteFile(filepath.Dir(app.Path)+"/apollo.config."+namespace, []byte{}, 0644)
			if err1 != nil {
				panic(fmt.Errorf("写入文件失败 %v", err1))
			}
			err2 := watcher.Add(filepath.Dir(app.Path) + "/apollo.config." + namespace)
			if err2 != nil {
				Logger.Fatal(err2)
			}
			wg.Add(1)
			go func(path, appId, namespace string, ctx context.Context) {
				switch Conf.Type {
				case POLLING:
					UpdateViaHttpPolling(HttpReqConfig{
						Path:      path,
						AppId:     appId,
						Namespace: namespace,
					}, &wg, ctx)

					break
				case HOT:
					UpdateEnvViaHttpLongPolling(HttpReqConfig{
						Path:          path,
						AppId:         appId,
						Namespace:     namespace,
						ReleaseKey:    "",
						Notifications: "",
					}, &wg, ctx)
					break
				default:
					Logger.Error("配置文件 type 类型错误")
					panic(fmt.Errorf("配置文件 type 类型错误 %v", Conf.Type))
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
			Logger.Error("apollo 客户端正在停止")
			cancel()
		}
	}
}

func updateAppEnvironment(path string, namespaces []string) {
	var contents []byte

	for _, namespace := range namespaces {
		content, _ := ioutil.ReadFile(path + "/apollo.config." + namespace)
		if len(content) == 0 {
			continue
		}

		contents = append(contents, content...)
	}

	// 写入新 env 前会清空之前的 env
	err := ioutil.WriteFile(path+"/.env", contents, 0644)

	if err != nil {
		Logger.Fatal(err)
	}

	reloadSupervisor()
}

func listenAppNamespaceConfig(watcher *fsnotify.Watcher, path string, namespace []string) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if (event.Op&fsnotify.Chmod == fsnotify.Chmod) || (event.Op&fsnotify.Write == fsnotify.Write) {
				Logger.Info("配置更新", event)
				updateAppEnvironment(path, namespace)
			}
		case err, ok := <-watcher.Errors:
			Logger.Error("文件 watcher 过程中发生错误", err)
			if !ok {
				return
			}
		}
	}
}

func reloadSupervisor() {
	if _, err := exec.LookPath("supervisorctl"); err != nil {
		Logger.Info("当前环境中不存在 supervisor")
		return
	}

	c := exec.Command("supervisorctl", "reload", "all")

	if err := c.Run(); err != nil {
		Logger.Fatal("supervisor 重新加载失败")
	}
}

package main

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/lapollo/client"
)

var (
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	Version   string
	Branch    string
	Commit    string
	BuildTime string
	lowercase string
)

const (
	POLLING = iota
	HOT
)

var mutex = &sync.RWMutex{}

func main() {
	versionFlag := flag.Bool("version", false, "print the version")
	flag.Parse()

	if *versionFlag {
		log.Printf("Version: %s\n", Version)
		log.Printf("Branch: %s\n", Branch)
		log.Printf("Commit: %s\n", Commit)
		log.Printf("BuildTime: %s\n", BuildTime)
		log.Printf("lowercase: %s\n", lowercase)
		os.Exit(0)
	}

	client.Logger.Info("apollo 客户端启动")
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		client.Logger.Error("添加文件 watcher 失败", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	continuousUpdate(ctx, watcher)
	go handleSignal()
	wg.Wait()
	client.Logger.Info("apollo 客户端退出")
	defer func() {
		if err := watcher.Close(); err != nil {
			client.Logger.Error("Watch 关闭异常", err.Error())
		}
		stackTraceWhenPanic()
	}()
}

func continuousUpdate(ctx context.Context, watcher *fsnotify.Watcher) {
	for _, app := range client.Conf.Apps {
		go listenAppNamespaceConfig(watcher, filepath.Dir(app.Path), app.Namespace)
		for _, namespace := range app.Namespace {
			err1 := ioutil.WriteFile(filepath.Dir(app.Path)+"/apollo.config."+namespace, []byte{}, 0644)
			if err1 != nil {
				client.Logger.Errorf("写入文件失败 %v", err1)
			}
			err2 := watcher.Add(filepath.Dir(app.Path) + "/apollo.config." + namespace)
			if err2 != nil {
				client.Logger.Fatal("apollo.config"+namespace+"监控失败", err2)
			}
			wg.Add(1)
			go func(path, appId, namespace string, ctx context.Context) {
				switch client.Conf.Type {
				case POLLING:
					client.UpdateViaHttpPolling(client.HttpReqConfig{
						Path:      path,
						AppId:     appId,
						Namespace: namespace,
					}, &wg, ctx)

					break
				case HOT:
					client.UpdateEnvViaHttpLongPolling(client.HttpReqConfig{
						Path:          path,
						AppId:         appId,
						Namespace:     namespace,
						ReleaseKey:    "",
						Notifications: "",
					}, &wg, ctx)
					break
				default:
					client.Logger.Error("配置文件 type 类型错误")
				}
				return

			}(app.Path, app.AppId, namespace, ctx)
		}
	}
}

func handleSignal() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		switch <-signals {
		case syscall.SIGTERM, syscall.SIGQUIT:
			client.Logger.Error("apollo 客户端正在停止")
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
		client.Logger.Fatal(".env 更新失败。", err)
	}
	mutex.Lock()
	reloadSupervisor()
	mutex.Unlock()
}

func listenAppNamespaceConfig(watcher *fsnotify.Watcher, path string, namespace []string) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				client.Logger.Info("配置更新", event)
				updateAppEnvironment(path, namespace)
			}
		case err, ok := <-watcher.Errors:
			client.Logger.Error("文件 watcher 过程中发生错误", err)
			if !ok {
				return
			}
		}
	}
}

func reloadSupervisor() {
	if _, err := os.Stat("/var/run/supervisor.sock"); errors.Is(err, os.ErrNotExist) {
		client.Logger.Info("当前环境中不存在 supervisor")
		return
	}
	c := exec.Command("supervisorctl", "reload")
	if err := c.Run(); err != nil {
		client.Logger.Error("supervisor reload failed.", err.Error())
	} else {
		client.Logger.Info("supervisor reload success.")
	}
}

func stackTraceWhenPanic() {
	if e := recover(); e != nil {
		client.Logger.Error("lapollo-client 异常退出. Call Stack:", string(debug.Stack()))
	}
}

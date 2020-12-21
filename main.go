package main

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
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
	logger *log.Logger
	conf   *config
)

const (
	POLLING = iota
	HOT
)

func init() {
	conf = getConfigViaOsEnvironment()
	if conf == nil {
		conf = getConfigViaYaml()
	}

	if conf == nil {
		panic("获取不到 Apollo-Client 的配置文件 \n")
	}

	logger = newLogger()
}

func main() {
	logger.Info("Apollo-Client Start Work")
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		logger.Error("Apollo-Client 启动文件监听失败", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	keepUpdate(ctx, watcher)
	go handleSignal()

	wg.Wait()
	defer watcher.Close()
	logger.Info("Apollo-Client Stop Work")
}

func keepUpdate(ctx context.Context, watcher *fsnotify.Watcher) {
	for _, app := range conf.Apps {
		go listenAppNamespaceConfig(watcher, filepath.Dir(app.Path), app.Namespace, ctx)
		for _, namespace := range app.Namespace {
			if err := ioutil.WriteFile(filepath.Dir(app.Path)+"/apollo.config."+namespace, []byte{}, 0644); err != nil {
				panic(fmt.Errorf("写入文件失败 %v", err))
			}

			if err := watcher.Add(filepath.Dir(app.Path) + "/apollo.config." + namespace); err != nil {
				panic(fmt.Errorf("Apollo-Client 文件监听失败 %v", err))
			}

			wg.Add(1)

			func(path, appId, namespace string, ctx context.Context) {
				switch conf.Type {
				case POLLING:
					updateViaHttpPolling(HttpReqConfig{
						Path:      path,
						AppId:     appId,
						Namespace: namespace,
					}, &wg, ctx)
				case HOT:
					updateEnvViaHttpLongPolling(HttpReqConfig{
						Path:          path,
						AppId:         appId,
						Namespace:     namespace,
						ReleaseKey:    "",
						Notifications: "",
					}, &wg, ctx)
				default:
					logger.Error("配置文件 type 类型错误")
					panic(fmt.Errorf("配置文件 type 类型错误 %v", conf.Type))
				}
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
			logger.Info("Apollo-Client 正在停止运行")
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
		logger.Fatal(err)
	}

	reloadSupervisor()
}

func listenAppNamespaceConfig(watcher *fsnotify.Watcher, path string, namespace []string, ctx context.Context) {
	defer func() {
		fmt.Println("exit listenAppNamespaceConfig")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if (event.Op&fsnotify.Chmod == fsnotify.Chmod) || (event.Op&fsnotify.Write == fsnotify.Write) {
				logger.Info("配置更新", event)
				updateAppEnvironment(path, namespace)
			}
		case err, ok := <-watcher.Errors:
			logger.Error("文件 watcher 过程中发生错误", err)
			if !ok {
				return
			}
		}
	}
}

func reloadSupervisor() {
	if _, err := exec.LookPath("supervisorctl"); err != nil {
		logger.Info("当前环境中不存在 supervisor")
		return
	}

	c := exec.Command("supervisorctl", "reload")
		if err := c.Run(); err != nil {
		logger.Errorf("supervisor 重新加载失败 %v", err)
	}
}

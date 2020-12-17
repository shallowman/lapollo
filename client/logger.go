package client

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

var (
	path       = "/var/www/storage/logs/"
	name       = "lapollo-client"
	ext        = "log"
	timeFormat = "2006-01-02"
)

var Logger = newLogger()

func newLogger() *log.Logger {
	logPath := os.Getenv("APOLLO_CLIENT_LOG_PATH")

	if logPath == "" {
		logPath = path
	}

	logName := generateLogFilename(logPath, name, timeFormat, ext)
	file, err := os.OpenFile(logName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	if err != nil {
		return &log.Logger{
			Out:   os.Stdout,
			Hooks: nil,
			Formatter: &log.TextFormatter{
				DisableColors: true,
				FullTimestamp: true,
			},
			ReportCaller: false,
			Level:        log.InfoLevel,
			ExitFunc:     nil,
		}
	}

	return &log.Logger{
		Out:   file,
		Hooks: nil,
		Formatter: &log.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		},
		ReportCaller: false,
		Level:        log.InfoLevel,
		ExitFunc:     nil,
	}

}

func generateLogFilename(path string, name string, format string, ext string) string {
	return fmt.Sprintf("%s%s-%s.%s", path, name, time.Now().Format(format), ext)
}

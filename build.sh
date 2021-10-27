#!/usr/bin/env zsh
prog=lapollo
version=1.0.0
lowercase=ok

# 交叉编译
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -ldflags "\
-X main.Version=$version \
-X main.Branch=`git rev-parse --abbrev-ref HEAD` \
-X main.Commit=`git rev-parse HEAD` \
-X main.BuildTime=`date -u '+%Y-%m-%d_%H:%M:%S'` \
-X main.lowercase=$lowercase \
" -v -o $prog main.go
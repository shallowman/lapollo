# lapollo 使用文档

## lapollo 是什么

lapollo 是由 go 语言开发的用于实时更新 [Laravel](https://laravel.com/) 框架（一个使用 php 开发的 web 框架）.env 环境变量文件的 apollo 客户端。

## 如何使用
### 1. 使用 Go 编译出当前操作系统下的可执行文件
- 使用 go build 构建出 Linux 下的可执行文件
```shell script
go build -o lapollo main.go
```
### 2. 配置文件初始化
#### 2.1 通过系统变量来设置客户端的配置
```shell
#Cluster  
APOLLO_CLUSTER  
# apollo-server host   
APOLLO_HOST 
# 应用的 .env 文件所在路径，绝对路径 
APOLLO_ENV_PATH 
# app id   
APOLLO_APP_ID   
# namespace    
APOLLO_NAMESPACE    
# 客户端日志路径  
APOLLO_CLIENT_LOG_PATH  
```
#### 2.2 通过 app.yaml 文件来设置客户端配置
- 修改 app.yaml 文件，将应用 id 等配置项替换成实际用到的值
- 执行 lapollo

```shell script
./lapollo
```
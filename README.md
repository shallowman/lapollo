# lapollo 使用文档

## lapollo 是什么

lapollo 是由 go 语言开发的用于实时更新 [Laravel](https://laravel.com/) 框架（一个使用 php 开发的 web 框架）.env 环境变量文件的 apollo 客户端。

## 如何使用
- 使用 go build 编程成 Linux 下的可执行文件
```shell script
go build -o lapollo main.go
```
- 赋予生成的可执行文件执行权限
```shell script
chmod +x lapollo
```
- 修改 app.yaml 文件，将应用 id 等配置项替换成实际用到的值
- 执行 lapollo
```shell script
./lapollo
```
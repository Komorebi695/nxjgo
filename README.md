# NxjGo Web Framework
## 介绍
NxjGo是一个用Go（Golang）编写的HTTP web框架。它使用类似于Martini的API，性能不一定很好，速度也不一定很快。这只是一个普普通通的框架。简单豁达，从容生活。

**NxjGo的主要特点是：**

- 中间件支持
- 路由分组
- 日志管理
- 错误管理
- 内置渲染
- orm支持
- 拓展
## 快速开始
### 要求

- Go: 1.20 及以上版本
### 获取NxjGo
使用 [Go 模块](https://github.com/golang/go/wiki/Modules)支持，只需添加以下导入
```go
import "github.com/Komorebi695/nxjgo"
```
到你的代码，然后会自动获取必要的依赖项。
否则，请运行以下 Go 命令来安装软件包：nxjgo
```
$ go get -u github.com/Komorebi695/nxjgo
```
### 运行NxjGo
首先，您需要导入NxjGo软件包以使用NxjGo，一个最简单的示例如下：
```go
package main

import (
	"github.com/Komorebi695/nxjgo"
	"net/http"
)

func main() {
	r := nxjgo.Default()
	g := r.Group("hello")
	g.Get("/ping", func(ctx *nxjgo.Context) {
		ctx.JSON(http.StatusOK, "Hello NxjGo!")
	})
	r.Run(":8080")
}

```
使用 Go 命令运行演示：
```
$ go run main.go
```

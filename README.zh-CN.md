# guacamole-common-go

[English](README.md) | 中文

Apache Guacamole `guacamole-common` 库的 Go 移植，让 Go 服务通过 guacd 为浏览器代理 RDP/VNC/SSH/Telnet 远程会话。

## 特性

- 完整的 Guacamole 协议实现：按 Unicode 码点计长的指令编解码、流式解析器、完整的连接握手（协议版本 1.0.0–1.5.0）
- 兼容 [guacamole-common-js](https://www.npmjs.com/package/guacamole-common-js) 的 WebSocket 隧道处理器：子协议协商、UUID 帧、ping 处理、指令批量转发
- 实现 `connect` / `read` / `write` 长轮询协议的 HTTP 隧道处理器，带会话令牌与隧道自动清理
- guacd 连接的 Socket 层：明文 TCP、TLS，以及能在首个 sync 之前暴露上游错误的 failover socket
- 基于 UUID 会话跟踪、读写独占访问的 Tunnel
- 指令过滤器，可在任一方向拦截、改写或丢弃指令
- API 对齐 Java `guacamole-common`（含 `Get*` 别名），便于移植
- 提供 RDP、VNC、SSH、Telnet、Kubernetes 可运行示例，均内嵌 Web 界面

## 快速开始

需要一个运行中的 [guacd](https://guacamole.apache.org/doc/gug/guacamole-architecture.html)（默认端口 4822）。

```bash
go get github.com/hq0101/guacamole-common-go
```

提供一个 WebSocket 隧道，供浏览器端 `guacamole-common-js` 客户端连接：

```go
package main

import (
	"context"
	"log"
	"net/http"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
	guacws "github.com/hq0101/guacamole-common-go/websocket"
)

func main() {
	connect := func(r *http.Request) (guacnet.Tunnel, error) {
		cfg := &protocol.Config{
			Protocol: "rdp",
			Params: map[string]string{
				"hostname":    "192.168.1.100",
				"port":        "3389",
				"username":    "alice",
				"password":    "secret",
				"ignore-cert": "true",
			},
		}
		return guacnet.Connect(context.Background(), "localhost:4822", cfg, protocol.NewClientInfo())
	}

	http.Handle("/ws", guacws.NewHandler(connect))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

浏览器端用 `new Guacamole.WebSocketTunnel("ws://localhost:8080/ws")` 创建 `Guacamole.Client` 即可——完整可运行的前端见 `examples/*/static/index.html` 内嵌页面。

如需使用 HTTP 长轮询隧道代替 WebSocket，把同一个 `connect` 函数传给 `servlet.NewHandler`：

```go
handler := servlet.NewHandler(connect) // 导入 "github.com/hq0101/guacamole-common-go/servlet"
defer handler.Shutdown()
http.Handle("/tunnel", handler)
```

连接参数也可以像示例那样，在 `connect` 内从请求（查询字符串、请求头、会话）中获取。设置 `Config.ConnectionID` 可加入已有的 guacd 连接，而不是新建连接。

## 示例

每个示例都内嵌 Web 界面，支持 `-guacd host:port` 和 `-addr` 参数：

| 示例 | 协议 | 默认端口 |
|---|---|---|
| `examples/rdp` | RDP | `:8083` |
| `examples/ssh` | SSH | `:8082` |
| `examples/telnet` | Telnet | `:8084` |
| `examples/vnc` | VNC | `:8085` |
| `examples/kubernetes` | Kubernetes exec | `:8086` |

```bash
go run ./examples/rdp -guacd localhost:4822
```

然后打开 http://localhost:8083 填写连接表单。连接参数在浏览器表单中输入；guacd 地址默认为局域网地址，请显式传 `-guacd`。

## 包结构

| 包 | 职责 |
|---|---|
| `protocol` | 指令、解析器、连接配置、客户端信息、状态码、协议版本 |
| `io` | `GuacamoleReader` / `GuacamoleWriter` 指令流及过滤器 |
| `net` | guacd socket（TCP/TLS/failover）、握手、UUID 隧道 |
| `websocket` | 面向 guacamole-common-js 的 WebSocket 隧道 `http.Handler` |
| `servlet` | 带会话管理的 HTTP 长轮询隧道处理器 |

## 许可

Apache License 2.0 — 见 [LICENSE](LICENSE)。
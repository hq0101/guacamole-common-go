// RDP example: serves a browser-based RDP session via guacd and guacamole-common-js.
// Connection parameters are provided through the web form in the browser.
//
// Usage:
//
//	go run . -guacd localhost:4822
package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	guacnet "github.com/hq0101/guacamole-common-go/net"
	"github.com/hq0101/guacamole-common-go/protocol"
	guacws "github.com/hq0101/guacamole-common-go/websocket"
)

//go:embed static
var staticFS embed.FS

func main() {
	guacd := flag.String("guacd", "192.168.127.131:4822", "guacd address (host:port)")
	addr := flag.String("addr", ":8083", "HTTP listen address")
	flag.Parse()

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", http.FileServer(http.FS(static)))
	http.Handle("/ws", guacws.NewHandler(func(r *http.Request) (guacnet.Tunnel, error) {
		q := r.URL.Query()
		port, _ := strconv.Atoi(q.Get("port"))
		width := intParam(q, "width", 1024)
		height := intParam(q, "height", 768)
		dpi := intParam(q, "dpi", 96)

		cfg := &protocol.Config{
			Protocol: "rdp",
			Params: map[string]string{
				"hostname":     q.Get("host"),
				"port":         strconv.Itoa(port),
				"username":     q.Get("user"),
				"password":     q.Get("password"),
				"domain":       q.Get("domain"),
				"width":        strconv.Itoa(width),
				"height":       strconv.Itoa(height),
				"color-depth":  q.Get("color-depth"),
				"security":     q.Get("security"),
				"ignore-cert":  q.Get("ignore-cert"),
				"disable-auth": q.Get("disable-auth"),

				// Visual quality: show wallpaper and window theming.
				"enable-wallpaper": "true",
				"enable-theming":   "true",
			},
		}
		info := &protocol.ClientInfo{
			OptimalWidth:      width,
			OptimalHeight:     height,
			OptimalResolution: dpi,
			Timezone:          q.Get("timezone"),
			ImageMimetypes:    []string{"image/png", "image/jpeg", "image/webp"},
		}
		return guacnet.Connect(context.Background(), *guacd, cfg, info)
	}))

	log.Printf("RDP example: open http://localhost%s in your browser", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func intParam(values mapParam, name string, fallback int) int {
	value, err := strconv.Atoi(values.Get(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

type mapParam interface {
	Get(string) string
}

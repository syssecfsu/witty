package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var host *string = nil

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		org := r.Header.Get("Origin")
		h, err := url.Parse(org)

		if err != nil {
			return false
		}

		if (host == nil) || (*host != h.Host) {
			fmt.Println("failed origin check of ", org, "against", *host)
		}

		return (host != nil) && (*host == h.Host)
	},
}

// handle websockets
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Created the websocket")

	for {
		msgType, p, err := conn.ReadMessage()

		if err != nil {
			fmt.Println(err)
			return
		}

		if err := conn.WriteMessage(msgType, p); err != nil {
			fmt.Println(err)
			return
		}
	}
}

// return files
func fileHandler(c *gin.Context, fname string) {
	// if the URL has no fname, c.Param returns "/"
	if fname == "/" {
		fname = "/index.html"
		host = &c.Request.Host
	}

	fname = fname[1:] //fname always starts with /
	fmt.Println(fname)

	if strings.HasSuffix(fname, "html") {
		c.HTML(200, fname, nil)
	} else {
		//c.HTML interprets the file as HTML file
		//we do not need that for regular files
		c.File(fmt.Sprint("assets/", fname))
	}
}

func main() {
	rt := gin.Default()

	rt.SetTrustedProxies(nil)
	rt.LoadHTMLGlob("assets/*.html")

	rt.GET("/*fname", func(c *gin.Context) {
		fname := c.Param("fname")

		// ws is a special case to create a new websocket
		switch fname {
		case "/ws":
			wsHandler(c.Writer, c.Request)
		default:
			fileHandler(c, fname)
		}
	})

	rt.Run(":8080")
}

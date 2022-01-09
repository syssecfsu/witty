package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// command line options
var cmdToExec = []string{"bash"}

func main() {
	fp, err := os.OpenFile("web_term.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)

	if err == nil {
		defer fp.Close()
		log.SetOutput(fp)
		gin.DefaultWriter = fp
	}

	// parse the arguments. User can pass the command to execute
	// by default, we use bash, but macos users might want to use zsh
	// you can also run single program, such as pstree, htop...
	// but program might misbehave (htop seems to be fine)
	args := os.Args

	if len(args) > 1 {
		cmdToExec = args[1:]
		log.Println(cmdToExec)
	}

	rt := gin.Default()

	rt.SetTrustedProxies(nil)
	rt.LoadHTMLGlob("./assets/*.html")

	rt.GET("/watch/*sname", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Watcher terminal",
			"path":  "/ws_watch",
		})
	})

	rt.GET("/ws_run", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request, false)
	})

	rt.GET("/ws_watch", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request, true)
	})

	// handle static files
	rt.Static("/assets", "./assets")

	rt.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Master terminal",
			"path":  "/ws_run ",
		})
		host = &c.Request.Host
	})

	rt.RunTLS(":8080", "./tls/cert.pem", "./tls/private-key.pem")
}

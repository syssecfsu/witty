package main

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/syssecfsu/witty/term_conn"
)

// command line options
var cmdToExec = []string{"bash"}

var host *string = nil

// simple function to check origin
func checkOrigin(r *http.Request) bool {
	org := r.Header.Get("Origin")
	h, err := url.Parse(org)

	if err != nil {
		return false
	}

	if (host == nil) || (*host != h.Host) {
		log.Println("Failed origin check of ", org)
	}

	return (host != nil) && (*host == h.Host)
}

type InteractiveSession struct {
	Ip  string
	Cmd string
	Id  string
}

func fillIndex(c *gin.Context) {
	var players []InteractiveSession

	term_conn.ForEachSession(func(tc *term_conn.TermConn) {
		players = append(players, InteractiveSession{
			Id:  tc.Name,
			Ip:  tc.Ip,
			Cmd: cmdToExec[0],
		})
	})

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":   "interactive terminal",
		"players": players,
	})
}

func main() {
	fp, err := os.OpenFile("witty.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)

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

	// Fill in the index page
	rt.GET("/", func(c *gin.Context) {
		host = &c.Request.Host
		fillIndex(c)
	})

	// create a new interactive session
	rt.GET("/new", func(c *gin.Context) {
		if host == nil {
			host = &c.Request.Host
		}

		id := uniuri.New()

		c.HTML(http.StatusOK, "term.html", gin.H{
			"title": "interactive terminal",
			"path":  "/ws_new/" + id,
			"id":    id,
		})
	})

	rt.GET("/ws_new/:id", func(c *gin.Context) {
		id := c.Param("id")
		term_conn.ConnectTerm(c.Writer, c.Request, false, id, cmdToExec)
	})

	// create a viewer of an interactive session
	rt.GET("/view/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.HTML(http.StatusOK, "term.html", gin.H{
			"title": "viewer terminal",
			"path":  "/ws_view/" + id,
		})
	})

	rt.GET("/ws_view/:id", func(c *gin.Context) {
		id := c.Param("id")
		term_conn.ConnectTerm(c.Writer, c.Request, true, id, nil)
	})

	// start/stop recording the session
	rt.GET("/record/:id", func(c *gin.Context) {
		id := c.Param("id")
		term_conn.StartRecord(id)
	})

	rt.GET("/stop/:id", func(c *gin.Context) {
		id := c.Param("id")
		term_conn.StopRecord(id)
	})

	// create a viewer of an interactive session
	rt.GET("/replay/*id", func(c *gin.Context) {
		id := c.Param("id")
		log.Println("replay/ called with", id)
		c.HTML(http.StatusOK, "replay.html", nil)
	})

	// handle static files
	rt.Static("/assets", "./assets")

	term_conn.Init(checkOrigin)

	rt.RunTLS(":8080", "./tls/cert.pem", "./tls/private-key.pem")
}

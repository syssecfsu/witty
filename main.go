package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

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

type RecordedSession struct {
	Fname    string
	Fsize    string
	Duration string
	Time     string
}

// how many seconds of the session
func getDuration(fname string) int64 {
	fp, err := os.Open("./records/" + fname)

	if err != nil {
		log.Println("Failed to open record file", err)
		return 0
	}

	decoder := json.NewDecoder(fp)

	if decoder == nil {
		log.Println("Failed to create JSON decoder")
		return 0
	}

	// To work with javascript decoder, we organize the file as
	// an array of writeRecord. golang decode instead decode
	// as individual record. Call decoder.Token to skip opening [
	decoder.Token()

	var dur int64 = 0

	for decoder.More() {
		var record term_conn.WriteRecord

		if err := decoder.Decode(&record); err != nil {
			log.Println("Failed to decode record", err)
			continue
		}

		dur += record.Dur.Milliseconds()
	}

	return dur/1000 + 1
}

func collectTabData(c *gin.Context) (players []InteractiveSession, records []RecordedSession) {
	term_conn.ForEachSession(func(tc *term_conn.TermConn) {
		players = append(players, InteractiveSession{
			Id:  tc.Name,
			Ip:  tc.Ip,
			Cmd: cmdToExec[0],
		})
	})

	files, err := ioutil.ReadDir("./records/")

	if err == nil {
		for _, finfo := range files {
			fname := finfo.Name()
			if !strings.HasSuffix(fname, ".rec") {
				continue
			}
			fsize := finfo.Size() / 1024
			duration := getDuration(fname)

			records = append(records,
				RecordedSession{
					Fname:    fname,
					Fsize:    strconv.FormatInt(fsize, 10),
					Duration: strconv.FormatInt(duration, 10),
					Time:     finfo.ModTime().Format("Jan/2/2006, 15:04:05"),
				})
		}
	}

	return
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
	rt.LoadHTMLGlob("./assets/template/*")

	// Fill in the index page
	rt.GET("/", func(c *gin.Context) {
		host = &c.Request.Host
		players, records := collectTabData(c)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":   "interactive terminal",
			"players": players,
			"records": records,
		})
	})

	rt.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./assets/img/favicon.ico")
	})

	// to update the tabs of current interactive and saved sessions
	rt.GET("/update/:active", func(c *gin.Context) {
		var active0, active1 string

		// setup which tab is active, it is hard to do in javascript at
		// client side due to timing issues.
		which := c.Param("active")
		if which == "0" {
			active0 = "active"
			active1 = ""
		} else {
			active0 = ""
			active1 = "active"
		}

		host = &c.Request.Host
		players, records := collectTabData(c)

		c.HTML(http.StatusOK, "tab.html", gin.H{
			"players": players,
			"records": records,
			"active0": active0,
			"active1": active1,
		})
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
			"logo":  "keyboard",
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
			"id":    id,
			"logo":  "view",
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
	rt.GET("/replay/:id", func(c *gin.Context) {
		id := c.Param("id")
		log.Println("replay/ called with", id)
		c.HTML(http.StatusOK, "replay.html", gin.H{
			"fname": id,
		})
	})

	rt.GET("/delete/:fname", func(c *gin.Context) {
		fname := c.Param("fname")
		if err := os.Remove("./records/" + fname); err != nil {
			log.Println("Failed to delete file,", err)
		}
	})

	// handle static files
	rt.Static("/assets", "./assets")
	rt.Static("/records", "./records")

	term_conn.Init(checkOrigin)

	rt.RunTLS(":8080", "./tls/cert.pem", "./tls/private-key.pem")
}

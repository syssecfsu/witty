package web

import (
	"net/http"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/syssecfsu/witty/term_conn"
)

type InteractiveSession struct {
	Ip  string
	Cmd string
	Id  string
}

func collectSessions(c *gin.Context, cmd string) (players []InteractiveSession) {
	term_conn.ForEachSession(func(tc *term_conn.TermConn) {
		players = append(players, InteractiveSession{
			Id:  tc.Name,
			Ip:  tc.Ip,
			Cmd: cmd,
		})
	})

	return
}

func indexPage(c *gin.Context) {
	host = &c.Request.Host
	var disabled = ""

	if noAuth {
		disabled = "disabled"
	}

	c.HTML(http.StatusOK, "index.html", gin.H{"disabled": disabled})
}

func updateIndex(c *gin.Context) {
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

	players := collectSessions(c, cmdToExec[0])
	records := collectRecords(c, cmdToExec[0])

	c.HTML(http.StatusOK, "tab.html", gin.H{
		"players": players,
		"records": records,
		"active0": active0,
		"active1": active1,
	})
}

func newInteractive(c *gin.Context) {
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
}

func newTermConn(c *gin.Context) {
	id := c.Param("id")
	term_conn.ConnectTerm(c.Writer, c.Request, false, id, cmdToExec)
}

func viewPage(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "term.html", gin.H{
		"title": "viewer terminal",
		"path":  "/ws_view/" + id,
		"id":    id,
		"logo":  "view",
	})
}

func newViewWS(c *gin.Context) {
	id := c.Param("id")
	term_conn.ConnectTerm(c.Writer, c.Request, true, id, nil)
}

func favIcon(c *gin.Context) {
	c.File("./assets/img/favicon.ico")
}

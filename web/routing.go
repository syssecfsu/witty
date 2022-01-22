package web

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/syssecfsu/witty/term_conn"
)

var host *string = nil
var cmdToExec []string
var noAuth bool

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

func StartWeb(fp *os.File, cmd []string, naked bool) {
	cmdToExec = cmd
	noAuth = naked

	if fp != nil {
		gin.DefaultWriter = fp
	}

	rt := gin.Default()

	// We randomly generate a key for now, should use a fixed key
	// so login can survive server reboot
	store := sessions.NewCookieStore([]byte(uniuri.NewLen(32)))
	rt.Use(sessions.Sessions("witty-session", store))

	rt.SetTrustedProxies(nil)
	rt.LoadHTMLGlob("./assets/template/*")
	// handle static files
	rt.Static("/assets", "./assets")
	rt.Static("/records", "./records")
	rt.GET("/favicon.ico", favIcon)

	rt.GET("/login", loginPage)
	rt.POST("/login", login)

	g1 := rt.Group("/")

	if !naked {
		g1.Use(AuthRequired)
	}

	// Fill in the index page
	g1.GET("/", indexPage)
	g1.GET("/logout", logout)

	// to update the tabs of current interactive and saved sessions
	g1.GET("/update/:active", updateIndex)

	// create a new interactive session
	g1.GET("/new", newInteractive)
	g1.GET("/ws_new/:id", newTermConn)

	// create a viewer of an interactive session
	g1.GET("/view/:id", viewPage)
	g1.GET("/ws_view/:id", newViewWS)

	// start/stop recording the session
	g1.GET("/record/:id", startRecord)
	g1.GET("/stop/:id", stopRecord)

	// create a viewer of an interactive session
	g1.GET("/replay/:id", replayPage)

	// delete a recording
	g1.GET("/delete/:fname", delRec)

	term_conn.Init(checkOrigin)
	rt.RunTLS(":8080", "./tls/cert.pem", "./tls/private-key.pem")
}

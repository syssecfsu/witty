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

func StartWeb(fp *os.File, cmd []string) {
	cmdToExec = cmd

	if fp != nil {
		gin.DefaultWriter = fp
	}

	rt := gin.Default()

	// We randomly generate a key for now, should use a fixed key
	// so login can survive server reboot
	store := sessions.NewCookieStore([]byte(uniuri.NewLen(32)))
	rt.Use(sessions.Sessions("witty-session", store))
	rt.Use(AuthRequired)

	rt.SetTrustedProxies(nil)
	rt.LoadHTMLGlob("./assets/template/*")

	// Fill in the index page
	rt.GET("/", indexPage)
	rt.GET("/login", loginPage)

	rt.POST("/login", login)
	rt.GET("/logout", logout)

	// to update the tabs of current interactive and saved sessions
	rt.GET("/update/:active", updateIndex)

	// create a new interactive session
	rt.GET("/new", newInteractive)
	rt.GET("/ws_new/:id", newTermConn)

	// create a viewer of an interactive session
	rt.GET("/view/:id", viewPage)
	rt.GET("/ws_view/:id", newViewWS)

	// start/stop recording the session
	rt.GET("/record/:id", startRecord)
	rt.GET("/stop/:id", stopRecord)

	// create a viewer of an interactive session
	rt.GET("/replay/:id", replayPage)

	// delete a recording
	rt.GET("/delete/:fname", delRec)

	// handle static files
	rt.Static("/assets", "./assets")
	rt.Static("/records", "./records")
	rt.GET("/favicon.ico", favIcon)

	term_conn.Init(checkOrigin)
	rt.RunTLS(":8080", "./tls/cert.pem", "./tls/private-key.pem")
}

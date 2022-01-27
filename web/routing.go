package web

import (
	"os"
	"strconv"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	adapter "github.com/gwatts/gin-adapter"
	"github.com/syssecfsu/witty/term_conn"
)

var cmdToExec []string
var noAuth bool

func StartWeb(fp *os.File, cmd []string, naked bool, port uint16) {
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

	csrfHttp := csrf.Protect([]byte(uniuri.NewLen(32)), csrf.Path("/"))
	csrfGin := adapter.Wrap(csrfHttp)
	rt.Use(csrfGin)

	rt.SetTrustedProxies(nil)
	rt.LoadHTMLGlob("./assets/template/*")
	// handle static files
	rt.StaticFS("/assets", assetFS())
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
	g1.POST("/new", newInteractive)
	g1.GET("/ws_new/:id", newTermConn)

	// create a viewer of an interactive session
	g1.GET("/view/:id", viewPage)
	g1.GET("/ws_view/:id", newViewWS)

	// start/stop recording the session
	g1.POST("/record/:id", startRecord)
	g1.POST("/stop/:id", stopRecord)

	// create a viewer of an interactive session
	g1.GET("/replay/:id", replayPage)

	// delete a recording
	g1.POST("/delete/:fname", delRec)
	// Rename a recording
	g1.POST("/rename/:oldname/:newname", renameRec)

	term_conn.Init()
	rt.RunTLS(":"+strconv.FormatUint(uint64(port), 10), "./tls/cert.pem", "./tls/private-key.pem")
}

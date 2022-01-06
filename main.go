package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

func createPty(cmdline string) (*os.File, *term.State, error) {
	// Create a shell command.
	cmd := exec.Command(cmdline)

	// Start the command with a pty.
	ptmx, err := pty.Start(cmd)

	if err != nil {
		return nil, nil, err
	}

	// Use fixed size, the xterm is initalized as 122x37,
	// But we set pty to 120x36. Using fullsize will lead
	// some program to misbehaive.
	pty.Setsize(ptmx, &pty.Winsize{
		Cols: 120,
		Rows: 36,
	})

	// Set stdin in raw mode. This might cause problems in ssh.
	// ignore the error if it so happens
	termState, err := term.MakeRaw(int(os.Stdin.Fd()))

	if err != nil {
		fmt.Println(err)
		return ptmx, nil, err
	}

	return ptmx, termState, nil
}

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

	ptmx, termState, err := createPty("bash")

	defer func() {
		//close the terminal and restore the terminal state
		ptmx.Close()

		if termState != nil {
			term.Restore(int(os.Stdin.Fd()), termState)
		}
	}()

	if err != nil {
		fmt.Println("failed to create PTY", err)
		return
	}

	// pipe the msgs from WS to pty, we need to use goroutine here
	go func() {
		for {
			_, buf, err := conn.ReadMessage()

			if err != nil {
				fmt.Println(err)
				// We need to close pty so the goroutine and this one can end
				// using defer will cause problems
				ptmx.Close()
				return
			}

			_, err = ptmx.Write(buf)

			if err != nil {
				fmt.Println(err)
				ptmx.Close()
				return
			}
		}
	}()

	readBuf := make([]byte, 4096)

	for {
		n, err := ptmx.Read(readBuf)

		if err != nil {
			fmt.Println(err)
			ptmx.Close()
			return
		}

		if err = conn.WriteMessage(websocket.BinaryMessage, readBuf[:n]); err != nil {
			ptmx.Close()
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

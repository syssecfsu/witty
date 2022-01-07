package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 5 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 8192

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Time to wait before force close on connection.
	closeGracePeriod = 10 * time.Second
)

func createPty(cmdline []string) (*os.File, *exec.Cmd, error) {
	// Create a shell command.
	cmd := exec.Command(cmdline[0], cmdline[1:]...)

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

	log.Printf("Create shell process %v (%v)", cmdline, cmd.Process.Pid)
	return ptmx, cmd, nil
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
			log.Println("Failed origin check of ", org)
		}

		return (host != nil) && (*host == h.Host)
	},
}

// Periodically send ping message to detect the status of the ws
func ping(ws *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ws.WriteControl(websocket.PingMessage,
				[]byte{}, time.Now().Add(writeWait))

			if err != nil {
				log.Println("Failed to write ping message:", err)
			}

		case <-done:
			log.Println("Exit ping routine as stdout is going away")
			return
		}
	}
}

// shovel data from websocket to pty stdin
func toPtyStdin(ws *websocket.Conn, ptmx *os.File) {
	ws.SetReadLimit(maxMessageSize)

	// set the readdeadline. The idea here is simple,
	// as long as we keep receiving pong message,
	// the readdeadline will keep updating. Otherwise
	// read will timeout.
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, buf, err := ws.ReadMessage()

		if err != nil {
			log.Println("Failed to receive data from ws:", err)
			break
		}

		_, err = ptmx.Write(buf)

		if err != nil {
			log.Println("Failed to send data to pty stdin: ", err)
			break
		}
	}
}

// shovel data from pty Stdout to WS
func fromPtyStdout(ws *websocket.Conn, ptmx *os.File, done chan struct{}) {
	readBuf := make([]byte, 4096)

	for {
		n, err := ptmx.Read(readBuf)

		if err != nil {
			log.Println("Failed to read from pty stdout: ", err)
			break
		}

		ws.SetWriteDeadline(time.Now().Add(writeWait))
		if err = ws.WriteMessage(websocket.BinaryMessage, readBuf[:n]); err != nil {
			log.Println("Failed to write message: ", err)
			break
		}
	}

	close(done)

	ws.SetWriteDeadline(time.Now().Add(writeWait))
	ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Pty closed"))
	time.Sleep(closeGracePeriod)
}

var cmdToExec = []string{"bash"}

// handle websockets
func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Failed to create websocket: ", err)
		return
	}

	defer ws.Close()

	log.Println("\n\nCreated the websocket")

	ptmx, cmd, err := createPty(cmdToExec)

	if err != nil {
		log.Println("Failed to create PTY: ", err)
		return
	}

	done := make(chan struct{})

	go fromPtyStdout(ws, ptmx, done)
	go ping(ws, done)

	toPtyStdin(ws, ptmx)

	// cleanup the pty and its related process
	ptmx.Close()
	proc := cmd.Process

	// send an interrupt, this will cause the shell process to
	// return from syscalls if any is pending
	if err := proc.Signal(os.Interrupt); err != nil {
		log.Printf("Failed to send Interrupt to shell process(%v): %v ", proc.Pid, err)
	}

	// Wait for a second for shell process to interrupt before kill it
	time.Sleep(time.Second)

	log.Printf("Try to kill the shell process(%v)", proc.Pid)

	if err := proc.Signal(os.Kill); err != nil {
		log.Printf("Failed to send KILL to shell process(%v): %v", proc.Pid, err)
	}

	if _, err := proc.Wait(); err != nil {
		log.Printf("Failed to wait for shell process(%v): %v", proc.Pid, err)
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
	log.Println("Sending ", fname)

	if strings.HasSuffix(fname, "html") {
		c.HTML(200, fname, nil)
	} else {
		//c.HTML interprets the file as HTML file
		//we do not need that for regular files
		c.File(fmt.Sprint("./assets/", fname))
	}
}

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

	rt.RunTLS(":8080", "./tls/cert.pem", "./tls/private-key.pem")
}

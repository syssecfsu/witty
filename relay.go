//This file contains code to relay traffic between websocket and pty
package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
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

// TermConn represents the connected websocket and pty.
// if isViewer is true
type TermConn struct {
	ws   *websocket.Conn
	name string

	// only valid for doers
	ptmx  *os.File             // the pty that runs the command
	cmd   *exec.Cmd            // represents the process, we need it to terminate the process
	vchan chan *websocket.Conn // channel to receive viewers
	done  chan struct{}
}

func (tc *TermConn) createPty(cmdline []string) error {
	// Create a shell command.
	cmd := exec.Command(cmdline[0], cmdline[1:]...)

	// Start the command with a pty.
	ptmx, err := pty.Start(cmd)

	if err != nil {
		return err
	}

	// Use fixed size, the xterm is initalized as 122x37,
	// But we set pty to 120x36. Using fullsize will lead
	// some program to misbehave.
	pty.Setsize(ptmx, &pty.Winsize{
		Cols: 120,
		Rows: 36,
	})

	tc.ptmx = ptmx
	tc.cmd = cmd

	log.Printf("Create shell process %v (%v)", cmdline, cmd.Process.Pid)
	return nil
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
func (tc *TermConn) ping() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := tc.ws.WriteControl(websocket.PingMessage,
				[]byte{}, time.Now().Add(writeWait))

			if err != nil {
				log.Println("Failed to write ping message:", err)
			}

		case <-tc.done:
			log.Println("Exit ping routine as stdout is going away")
			return
		}
	}
}

// shovel data from websocket to pty stdin
func (tc *TermConn) wsToPtyStdin() {
	tc.ws.SetReadLimit(maxMessageSize)

	// set the readdeadline. The idea here is simple,
	// as long as we keep receiving pong message,
	// the readdeadline will keep updating. Otherwise
	// read will timeout.
	tc.ws.SetReadDeadline(time.Now().Add(pongWait))
	tc.ws.SetPongHandler(func(string) error {
		tc.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// we do not need to forward user input to viewers, only the stdout
	for {
		_, buf, err := tc.ws.ReadMessage()

		if err != nil {
			log.Println("Failed to receive data from ws:", err)
			break
		}

		_, err = tc.ptmx.Write(buf)

		if err != nil {
			log.Println("Failed to send data to pty stdin: ", err)
			break
		}
	}
}

// shovel data from pty Stdout to WS
func (tc *TermConn) ptyStdoutToWs() {
	var viewers []*websocket.Conn
	readBuf := make([]byte, 4096)

	for {
		n, err := tc.ptmx.Read(readBuf)

		if err != nil {
			log.Println("Failed to read from pty stdout: ", err)
			break
		}

		// handle viewers, we want to use non-blocking receive
		select {
		case watcher := <-tc.vchan:
			log.Println("Received watcher", watcher)
			viewers = append(viewers, watcher)
		default:
			log.Println("no watcher received")
		}

		// We could add ws to viewers as well (then we can use io.MultiWriter),
		// but we want to handle errors differently
		tc.ws.SetWriteDeadline(time.Now().Add(writeWait))
		if tc.ws.WriteMessage(websocket.BinaryMessage, readBuf[0:n]); err != nil {
			log.Println("Failed to write message: ", err)
			break
		}

		for i, w := range viewers {
			if w == nil {
				continue
			}

			// if the viewer exits, we will just ignore the error
			if err = w.WriteMessage(websocket.BinaryMessage, readBuf[0:n]); err != nil {
				log.Println("Failed to write message to watcher: ", err)

				viewers[i] = nil
				w.Close()
			}
		}
	}

	// close the watcher
	for _, w := range viewers {
		if w != nil {
			w.Close()
		}
	}

	tc.ws.SetWriteDeadline(time.Now().Add(writeWait))
	tc.ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Pty closed"))
	time.Sleep(closeGracePeriod)
}

func (tc *TermConn) release() {
	log.Println("releasing", tc.name)
	registry.delDoer(tc.name)

	if tc.ptmx != nil {
		// cleanup the pty and its related process
		tc.ptmx.Close()

		// terminate the command line process
		proc := tc.cmd.Process

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

		close(tc.done)
		close(tc.vchan)
	}

	tc.ws.Close()

}

// handle websockets
func wsHandleDoer(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Failed to create websocket: ", err)
		return
	}

	tc := TermConn{
		ws:   ws,
		name: "main",
	}

	defer tc.release()
	log.Println("\n\nCreated the websocket")

	if err := tc.createPty(cmdToExec); err != nil {
		log.Println("Failed to create PTY: ", err)
		return
	}

	tc.done = make(chan struct{})
	tc.vchan = make(chan *websocket.Conn)

	registry.addDoer("main", &tc)

	// main event loop to shovel data between ws and pty
	go tc.ping()
	go tc.wsToPtyStdin()

	tc.ptyStdoutToWs()
}

// handle websockets
func wsHandleViewer(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Failed to create websocket: ", err)
		return
	}

	log.Println("\n\nCreated the websocket")
	if !registry.sendToDoer("main", ws) {
		log.Println("Failed to send websocket to doer, close it")
		ws.Close()
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request, isViewer bool) {
	if !isViewer {
		wsHandleDoer(w, r)
	} else {
		wsHandleViewer(w, r)
	}
}

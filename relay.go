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
	var watchers []*websocket.Conn
	readBuf := make([]byte, 4096)

	for {
		n, err := ptmx.Read(readBuf)

		if err != nil {
			log.Println("Failed to read from pty stdout: ", err)
			break
		}

		// handle watchers, we want to use non-blocking receive
		select {
		case watcher := <-watcherChan:
			log.Println("Received watcher", watcher)
			watchers = append(watchers, watcher)

		default:
			log.Println("no watcher received")
		}

		// We could add ws to watchers as well, but we want to handle it
		// differently if there is an error
		ws.SetWriteDeadline(time.Now().Add(writeWait))
		if ws.WriteMessage(websocket.BinaryMessage, readBuf[0:n]); err != nil {
			log.Println("Failed to write message: ", err)
			break
		}

		for i, w := range watchers {
			if w == nil {
				continue
			}

			if err = w.WriteMessage(websocket.BinaryMessage, readBuf[0:n]); err != nil {
				log.Println("Failed to write message to watcher: ", err)

				watchers[i] = nil
				w.Close()
			}
		}
	}

	close(done)
	close(watcherChan)
	watcherChan = nil

	// close the watcher
	for _, w := range watchers {
		if w != nil {
			w.Close()
		}
	}

	ws.SetWriteDeadline(time.Now().Add(writeWait))
	ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Pty closed"))
	time.Sleep(closeGracePeriod)
}

var watcherChan chan *websocket.Conn

// handle websockets
func wsHandleRun(w http.ResponseWriter, r *http.Request) {
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
	watcherChan = make(chan *websocket.Conn)

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

// handle websockets
func wsHandleWatcher(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Failed to create websocket: ", err)
		return
	}

	log.Println("\n\nCreated the websocket")

	if watcherChan == nil {
		log.Println("No active runner, create a runner first")
		ws.Close()
		return
	}

	// hand the websocket to runner
	watcherChan <- ws
}

func wsHandler(w http.ResponseWriter, r *http.Request, isWatcher bool) {
	if !isWatcher {
		wsHandleRun(w, r)
	} else {
		wsHandleWatcher(w, r)
	}
}

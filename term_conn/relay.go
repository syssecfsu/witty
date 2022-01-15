//This file contains code to relay traffic between websocket and pty
package term_conn

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	readWait  = 10 * time.Second
	writeWait = 10 * time.Second
	viewWait  = 3 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize  = 4096
	readBufferSize  = 1024
	WriteBufferSize = 1024

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	recordCmd = 1
	stopCmd   = 0
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  readBufferSize,
	WriteBufferSize: WriteBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// TermConn represents the connected websocket and pty.
// if isViewer is true
type TermConn struct {
	Name string
	Ip   string

	ws          *websocket.Conn
	ptmx        *os.File             // the pty that runs the command
	record      *os.File             // record session
	lastRecTime time.Time            // last time a record is written
	cmd         *exec.Cmd            // represents the process, we need it to terminate the process
	viewChan    chan *websocket.Conn // channel to receive viewers
	recordChan  chan int             // channel to start/stop recording
	ws_done     chan struct{}        // ws is closed, only close this chan in ws reader
	pty_done    chan struct{}        // pty is closed, close this chan in pty reader
}

type writeRecord struct {
	Dur  time.Duration `json:"Duration"`
	Data []byte        `json:"Data"`
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

// Periodically send ping message to detect the status of the ws
func (tc *TermConn) ping(wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

out:
	for {
		select {
		case <-ticker.C:
			err := tc.ws.WriteControl(websocket.PingMessage,
				[]byte{}, time.Now().Add(writeWait))

			if err != nil {
				log.Println("Failed to write ping message:", err)
				break out
			}
		case <-tc.pty_done:
			log.Println("Exit ping routine as pty is going away")
			break out

		case <-tc.ws_done:
			log.Println("Exit ping routine as ws is going away")
			break out
		}
	}

	log.Println("Ping routine exited")
}

// shovel data from websocket to pty stdin
func (tc *TermConn) wsToPtyStdin(wg *sync.WaitGroup) {
	defer wg.Done()

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

	bufChan := make(chan []byte)

	go func() { //create a goroutine to read from ws
		for {
			_, buf, err := tc.ws.ReadMessage()

			if err != nil {
				log.Println("Failed to receive data from ws:", err)
				close(bufChan) // close chan by producer
				close(tc.ws_done)
				break
			}

			bufChan <- buf
		}
	}()
	// we do not need to forward user input to viewers, only the stdout
out:
	for {
		select {
		case buf, ok := <-bufChan:
			if !ok {
				log.Println("Exit wsToPtyStdin routine pty stdin error")
				break out
			}
			_, err := tc.ptmx.Write(buf)

			if err != nil {
				log.Println("Failed to send data to pty stdin: ", err)
				break out
			}
		case <-tc.ws_done:
			log.Println("Exit wsToPtyStdin routine as ws is going away")
			break out
		case <-tc.pty_done:
			log.Println("Exit wsToPtyStdin routine as pty is going away")
			break out
		}
	}

	log.Println("wsToPtyStdin routine exited")
}

// shovel data from pty Stdout to WS
func (tc *TermConn) ptyStdoutToWs(wg *sync.WaitGroup) {
	var viewers []*websocket.Conn

	defer wg.Done()
	bufChan := make(chan []byte)

	go func() { //create a goroutine to read from pty
		for {
			readBuf := make([]byte, 1024) //pty reads in 1024 blocks
			n, err := tc.ptmx.Read(readBuf)

			if err != nil {
				log.Println("Failed to read from pty stdout: ", err)
				close(bufChan)
				close(tc.pty_done)
				break
			}

			readBuf = readBuf[0:n] // slice the buffer so that it is exact the size of data read.
			bufChan <- readBuf
		}
	}()

out:
	for {
		// handle viewers, we want to use non-blocking receive
		select {
		case buf, ok := <-bufChan:
			if !ok {
				tc.ws.SetWriteDeadline(time.Now().Add(writeWait))
				tc.ws.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Pty closed"))

				break out
			}
			// We could add ws to viewers as well (then we can use io.MultiWriter),
			// but we want to handle errors differently
			tc.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := tc.ws.WriteMessage(websocket.BinaryMessage, buf); err != nil {
				log.Println("Failed to write message: ", err)
				break out
			}

			//write to the viewer
			for i, w := range viewers {
				if w == nil {
					continue
				}

				// if the viewer exits, we will just ignore the error
				w.SetWriteDeadline(time.Now().Add(viewWait))
				if err := w.WriteMessage(websocket.BinaryMessage, buf); err != nil {
					log.Println("Failed to write message to viewer: ", err)

					viewers[i] = nil
					w.Close() // we own the socket and need to close it
				}
			}

			// Do we need to record the session?
			if tc.record != nil {
				jbuf, err := json.Marshal(writeRecord{Dur: time.Since(tc.lastRecTime), Data: buf})
				if err != nil {
					log.Println("Failed to marshal record", err)
				} else {
					tc.record.Write(jbuf)
					tc.record.Write([]byte(",")) // write a deliminator
				}

				tc.lastRecTime = time.Now()
			}

		case cmd := <-tc.recordChan:
			var err error
			if cmd == recordCmd {
				// use the session ID and current as file name
				fname := "./records/" + tc.Name + "_" + strconv.FormatInt(time.Now().Unix(), 16) + ".rec"

				tc.record, err = os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					log.Println("Failed to create record file", fname, err)
					tc.record = nil
				}

				tc.record.Write([]byte("[")) // write a [ for an array of json objs
				tc.lastRecTime = time.Now()
			} else {
				fsinfo, err := tc.record.Stat()

				if err == nil {
					tc.record.Truncate(fsinfo.Size() - 1)
					tc.record.Seek(0, 2) // truncate does not change read/write location
					tc.record.Write([]byte("]"))
				}

				tc.record.Close()
				tc.record = nil
			}

		case viewer := <-tc.viewChan:
			log.Println("Received viewer", viewer.RemoteAddr().String())
			viewers = append(viewers, viewer)

		case <-tc.ws_done:
			log.Println("Exit ptyStdoutToWs routine as ws is going away")
			break out

		case <-tc.pty_done:
			log.Println("Exit ptyStdoutToWs routine as pty is going away")
			break out // do not block on these two channels
		}

	}

	// close the watcher
	for _, w := range viewers {
		if w != nil {
			w.Close()
		}
	}

	log.Println("ptyStdoutToWs routine exited")
}

// this function should be executed by the main goroutine for the connection
func (tc *TermConn) release() {
	log.Println("Releasing terminal connection", tc.Name)

	registry.removePlayer(tc.Name)

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

		close(tc.viewChan)
		close(tc.recordChan)
	}

	tc.ws.Close()
}

// handle websockets
func handlePlayer(w http.ResponseWriter, r *http.Request, name string, cmdline []string) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Failed to create websocket: ", err)
		return
	}

	tc := TermConn{
		ws:   ws,
		Name: name,
		Ip:   ws.RemoteAddr().String(),
	}

	defer tc.release()
	log.Println("Created the websocket to", ws.RemoteAddr().String())

	tc.ws_done = make(chan struct{})
	tc.pty_done = make(chan struct{})
	tc.viewChan = make(chan *websocket.Conn)
	tc.recordChan = make(chan int)

	if err := tc.createPty(cmdline); err != nil {
		log.Println("Failed to create PTY: ", err)
		return
	}

	registry.addPlayer(&tc)

	// main event loop to shovel data between ws and pty
	// do not call ptyStdoutToWs in this goroutine, otherwise
	// the websocket will not close. This is because ptyStdoutToWs
	// is usually blocked in the pty.Read
	var wg sync.WaitGroup
	wg.Add(3)

	go tc.ping(&wg)
	go tc.ptyStdoutToWs(&wg)
	go tc.wsToPtyStdin(&wg)

	wg.Wait()
	log.Println("Wait returned")
}

// handle websockets
func handleViewer(w http.ResponseWriter, r *http.Request, path string) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Failed to create websocket: ", err)
		return
	}

	log.Println("Created the websocket to", ws.RemoteAddr().String())
	if !registry.sendToPlayer(path, ws) {
		log.Println("Failed to send websocket to player, close it")
		ws.Close()
	}
}

func ConnectTerm(w http.ResponseWriter, r *http.Request, isViewer bool, name string, cmdline []string) {
	if !isViewer {
		handlePlayer(w, r, name, cmdline)
	} else {
		handleViewer(w, r, name)
	}
}

func Init(checkOrigin func(r *http.Request) bool) {
	upgrader.CheckOrigin = checkOrigin
	registry.init()
}

func StartRecord(id string) {
	registry.recordSession(id, recordCmd)
}

func StopRecord(id string) {
	registry.recordSession(id, stopCmd)
}

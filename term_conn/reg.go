package term_conn

import (
	"errors"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// a simple registry for actors and their channels. It is possible to
// design this using channels, but it is simple enough with mutex
type Registry struct {
	mtx     sync.Mutex
	players map[string]*TermConn
}

var registry Registry

func (reg *Registry) init() {
	reg.players = make(map[string]*TermConn)
}

func (d *Registry) addPlayer(tc *TermConn) {
	d.mtx.Lock()
	if _, ok := d.players[tc.Name]; ok {
		log.Println(tc.Name, "Already exist in the dispatcher, skip registration")
	} else {
		d.players[tc.Name] = tc
		log.Println("Add interactive session to registry", tc.Name)
	}
	d.mtx.Unlock()
}

func (d *Registry) removePlayer(name string) error {
	d.mtx.Lock()
	var err error = errors.New("not found")

	if _, ok := d.players[name]; ok {
		delete(d.players, name)
		err = nil
		log.Println("Removed interactive session to registry", name)
	}

	d.mtx.Unlock()
	return err
}

// we do not want to return the channel to viewer so it won't be used out of the critical section
func (d *Registry) sendToPlayer(name string, ws *websocket.Conn) bool {
	d.mtx.Lock()
	tc, ok := d.players[name]

	if ok {
		tc.vchan <- ws
	}

	d.mtx.Unlock()
	return ok
}

func ForEachSession(fp func(tc *TermConn)) {
	registry.mtx.Lock()
	for _, v := range registry.players {
		fp(v)
	}
	registry.mtx.Unlock()
}

package main

import (
	"errors"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// a simple registry for actors and their channels. It is possible to
// design this using channels, but it is simple enough with mutex
type Registry struct {
	mtx   sync.Mutex
	doers map[string]*TermConn
}

var registry Registry

func (reg *Registry) init() {
	reg.doers = make(map[string]*TermConn)
}

func (d *Registry) addDoer(name string, tc *TermConn) {
	d.mtx.Lock()
	if val, ok := d.doers[name]; ok {
		log.Println(name, "already exist in the dispatcher", val, tc)
		delete(d.doers, name)
		val.release(false) // do not unregister in release, otherwise it is a deadlock
	}
	d.doers[name] = tc
	d.mtx.Unlock()
}

func (d *Registry) delDoer(name string) error {
	d.mtx.Lock()
	var err error = errors.New("not found")

	if _, ok := d.doers[name]; ok {
		delete(d.doers, name)
		err = nil
	}

	d.mtx.Unlock()
	return err
}

// we do not want to return the channel to viewer so it won't be used out of the critical section
func (d *Registry) sendToDoer(name string, ws *websocket.Conn) bool {
	d.mtx.Lock()
	tc, ok := d.doers[name]

	if ok {
		tc.vchan <- ws
	}

	d.mtx.Unlock()
	return ok
}

package main

import (
	"net/http"
	"sync"
)

type replacingHandler struct {
	sync.RWMutex
	http.Handler
}

func (rh *replacingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rh.RLock()
	h := rh.Handler
	rh.RUnlock()
	h.ServeHTTP(w, r)
}

func (rh *replacingHandler) replace(h http.Handler) {
	rh.Lock()
	rh.Handler = h
	rh.Unlock()
}

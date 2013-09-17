package httpfstream

import (
	"code.google.com/p/go.tools/godoc/vfs/httpfs"
	"github.com/garyburd/go-websocket/websocket"
	"github.com/sourcegraph/rwvfs"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func New(root rwvfs.FileSystem) handler {
	return handler{
		Root:   root,
		httpFS: httpfs.New(root),
	}
}

type handler struct {
	Root rwvfs.FileSystem
	Log  *log.Logger

	httpFS http.FileSystem
}

const xMethod = "X-Method"

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "HEAD":
		h.Head(w, r)
	case "GET":
		switch r.Header.Get(xMethod) {
		case "GET":
			h.Get(w, r)
		case "PUT":
			h.Put(w, r)
		default:
			http.Error(w, xMethod+" value not supported", http.StatusBadRequest)
		}
	default:
		http.Error(w, "method not supported", http.StatusMethodNotAllowed)
	}
}

func (h handler) logf(msg string, v ...interface{}) {
	if h.Log != nil {
		h.Log.Printf(msg, v...)
	}
}

const (
	readBufSize  = 1024
	writeBufSize = 1024
)

var (
	readWait  = 1 * time.Second
	writeWait = 1 * time.Second
)

func (h handler) Head(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	f, err := h.Root.Open(r.URL.Path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "404 not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to open file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
}

func (h handler) Get(w http.ResponseWriter, r *http.Request) {
	if h.Log != nil {
		h.Log.Printf("GET %s", r.URL.Path)
	}

	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		http.Error(w, "path must not end with '/'", http.StatusBadRequest)
		return
	}

	f, err := h.Root.Open(r.URL.Path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "404 not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to open file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ws, err := websocket.Upgrade(w, r.Header, nil, readBufSize, writeBufSize)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			h.logf("not a WebSocket handshake: %s", err)
			return
		}
		h.logf("failed to upgrade to WebSocket: %s", err)
		return
	}
	defer ws.Close()

	for {
		w, err := ws.NextWriter(websocket.OpText)
		if err != nil {
			h.logf("NextWriter failed: %s", err)
			return
		}
		var n int64
		n, err = io.Copy(w, f)
		if err != nil {
			h.logf("Copy to WebSocket failed: %s", err)
			w.Close()
			return
		}

		err = w.Close()
		if err != nil {
			h.logf("Failed to close WebSocket: %s", err)
			return
		}

		h.logf("Read %d bytes from %s to WebSocket", n, r.URL.Path)
		if n == 0 {
			return
		}
	}

	if err != nil {
		http.Error(w, "failed to copy from file to response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = f.Close()
	if err != nil {
		http.Error(w, "failed to close destination file: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h handler) Put(w http.ResponseWriter, r *http.Request) {
	h.logf("PUT %s", r.URL.Path)

	defer r.Body.Close()

	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		http.Error(w, "path must not end with '/'", http.StatusBadRequest)
		return
	}

	f, err := h.Root.OpenFile(r.URL.Path, os.O_WRONLY|os.O_CREATE|os.O_APPEND)
	if err != nil {
		http.Error(w, "failed to open destination file for writing: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ws, err := websocket.Upgrade(w, r.Header, nil, readBufSize, writeBufSize)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			h.logf("not a WebSocket handshake: %s", err)
			return
		}
		h.logf("failed to upgrade to WebSocket: %s", err)
		return
	}
	defer ws.Close()

	for {
		ws.SetReadDeadline(time.Now().Add(readWait))
		op, rd, err := ws.NextReader()
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				h.logf("NextReader failed: %s", err)
			}
			break
		}
		if op != websocket.OpBinary && op != websocket.OpText {
			continue
		}

		n, err := io.Copy(f, rd)
		if err != nil {
			h.logf("Read from WebSocket failed: %s", err)
			return
		}

		h.logf("Read %d bytes from WebSocket to %s", n, r.URL.Path)

		if f, ok := f.(*os.File); ok {
			f.Sync()
		}
	}

	err = r.Body.Close()
	if err != nil {
		http.Error(w, "failed to close upload stream: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = f.Close()
	if err != nil {
		http.Error(w, "failed to close destination file: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

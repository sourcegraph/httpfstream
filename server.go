package httpfstream

import (
	"bytes"
	"errors"
	"github.com/garyburd/go-websocket/websocket"
	"io"
	"log"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"sync"
	"time"
)

// New returns a new http.Handler for httpfstream.
func New(root string) Handler {
	return Handler{
		Root:      root,
		httpFS:    http.Dir(root),
		writers:   make(map[string]struct{}),
		followers: make(map[string]map[*http.Request]chan []byte),
	}
}

type Handler struct {
	Root string
	Log  *log.Logger

	httpFS http.FileSystem

	writers   map[string]struct{}
	writersMu sync.Mutex

	followers   map[string]map[*http.Request]chan []byte
	followersMu sync.Mutex
}

const xVerb = "X-Verb"

// ServeHTTP implements net/http.Handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	verb := r.Header.Get(xVerb)
	if verb == "" {
		verb = r.URL.Query().Get("verb")
	}

	switch r.Method {
	case "GET":
		switch verb {
		case "APPEND":
			h.Append(w, r)
		default:
			h.Follow(w, r)
		}
	default:
		http.Error(w, "method not supported", http.StatusMethodNotAllowed)
	}
}

func (h Handler) logf(msg string, v ...interface{}) {
	if h.Log != nil {
		h.Log.Printf(msg, v...)
	}
}

const (
	readBufSize  = 10 * 1024 // 10 kb
	writeBufSize = 10 * 1024 // 10 kb

	writeChanSize = 50
)

var (
	followKeepaliveInterval = 3 * time.Second
	readWait                = 25 * time.Second
	writeWait               = 5 * time.Second
)

func (h Handler) resolve(path string) string {
	path = pathpkg.Clean("/" + path)
	return filepath.Join(string(h.Root), path)
}

// ErrWriterConflict indicates that the requested path is currently being
// written by another writer. A path may have at most one active writer.
var ErrWriterConflict = errors.New("path already has an active writer")

func (h Handler) addWriter(path string) error {
	h.writersMu.Lock()
	defer h.writersMu.Unlock()
	if _, present := h.writers[path]; !present {
		h.writers[path] = struct{}{}
		return nil
	}
	return ErrWriterConflict
}

func (h Handler) removeWriter(path string) {
	h.writersMu.Lock()
	defer h.writersMu.Unlock()
	delete(h.writers, path)
}

func (h Handler) addFollower(path string, r *http.Request, c chan []byte) {
	h.followersMu.Lock()
	defer h.followersMu.Unlock()
	if _, present := h.followers[path]; !present {
		h.followers[path] = make(map[*http.Request]chan []byte)
	}
	h.followers[path][r] = c
}

func (h Handler) getFollowers(path string) []chan []byte {
	h.followersMu.Lock()
	defer h.followersMu.Unlock()
	fs := make([]chan []byte, len(h.followers[path]))
	i := 0
	for _, f := range h.followers[path] {
		fs[i] = f
		i++
	}
	return fs
}

func (h Handler) removeFollower(path string, r *http.Request) {
	h.followersMu.Lock()
	defer h.followersMu.Unlock()
	delete(h.followers[path], r)
	if len(h.followers[path]) == 0 {
		delete(h.followers, path)
	}
}

func (h Handler) isWriting(path string) bool {
	h.writersMu.Lock()
	defer h.writersMu.Unlock()
	_, present := h.writers[path]
	return present
}

func (h Handler) serveFile(w http.ResponseWriter, r *http.Request) {
	http.FileServer(h.httpFS).ServeHTTP(w, r)
}

// Follow handles FOLLOW requests to retrieve the contents of a file and a
// real-time stream of data that is appended to the file.
func (h Handler) Follow(w http.ResponseWriter, r *http.Request) {
	path := h.resolve(r.URL.Path)
	h.logf("FOLLOW %s", path)

	// If this file isn't currently being written to, we don't need to update to
	// a WebSocket; we can just return the static file.
	if !h.isWriting(path) {
		h.serveFile(w, r)
		return
	}

	c := make(chan []byte)
	h.addFollower(path, r, c)
	defer h.removeFollower(path, r)

	// TODO(sqs): race conditions galore

	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "failed to open file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Open WebSocket.
	ws, err := websocket.Upgrade(w, r.Header, nil, readBufSize, writeBufSize)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			// Serve file via HTTP (not WebSocket).
			h.serveFile(w, r)
			return
		}
		h.logf("failed to upgrade to WebSocket: %s", err)
		return
	}
	defer ws.Close()

	// Send persisted file contents.
	for {
		sw, err := ws.NextWriter(websocket.OpText)
		if err != nil {
			h.logf("NextWriter for file failed: %s", err)
			return
		}

		n, err := io.Copy(sw, f)
		if err != nil {
			h.logf("File write to WebSocket failed: %s", err)
			sw.Close()
			return
		}

		err = sw.Close()
		if err != nil {
			h.logf("Failed to close WebSocket file writer: %s", err)
			return
		}

		// Finished reading file.
		if n == 0 {
			break
		}
	}

	// Follow new writes to file.
	var lastPing time.Time
	for {
		tick := time.NewTicker(50 * time.Millisecond)
		select {
		case <-tick.C:
			if !h.isWriting(path) {
				goto done
			}
			if time.Since(lastPing) > followKeepaliveInterval {
				ws.WriteMessage(websocket.OpPing, []byte{})
				lastPing = time.Now()
			}
		case data := <-c:
			sw, err := ws.NextWriter(websocket.OpText)
			if err != nil {
				h.logf("NextWriter failed: %s", err)
				return
			}

			_, err = sw.Write(data)
			if err != nil {
				h.logf("Write to WebSocket failed: %s", err)
				sw.Close()
				return
			}

			err = sw.Close()
			if err != nil {
				h.logf("Failed to close WebSocket writer: %s", err)
				return
			}
		}
	}

done:
	err = ws.WriteControl(websocket.OpClose, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
	if err != nil {
		h.logf("Failed to close WebSocket: %s", err)
		return
	}

	err = f.Close()
	if err != nil {
		h.logf("Failed to close destination file: %s", err)
		return
	}
}

// Append handles APPEND requests and appends data to a file.
func (h Handler) Append(w http.ResponseWriter, r *http.Request) {
	path := h.resolve(r.URL.Path)
	h.logf("APPEND %s", path)

	defer r.Body.Close()

	err := h.addWriter(path)
	if err != nil {
		h.logf("addWriter %s: %s", err)
		http.Error(w, "addWriter: "+err.Error(), http.StatusForbidden)
		return
	}
	defer h.removeWriter(path)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		http.Error(w, "failed to create dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
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

	ws.SetReadDeadline(time.Now().Add(readWait))
	for {
		op, rd, err := ws.NextReader()
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				h.logf("NextReader failed: %s", err)
			}
			break
		}
		switch op {
		case websocket.OpPong:
			ws.SetReadDeadline(time.Now().Add(readWait))
		case websocket.OpText:
			var buf bytes.Buffer
			mw := io.MultiWriter(f, &buf)

			// Persist to file.
			_, err := io.Copy(mw, rd)
			if err != nil {
				h.logf("Read from WebSocket failed: %s", err)
				return
			}

			// Broadcast to followers.
			followers := h.getFollowers(path)
			for _, fc := range followers {
				fc <- buf.Bytes()
			}
			ws.SetReadDeadline(time.Now().Add(readWait))
		}
	}

	err = r.Body.Close()
	if err != nil {
		h.logf("failed to close upload stream: %s", err)
		return
	}

	err = f.Close()
	if err != nil {
		h.logf("failed to close destination file: %s", err)
		return
	}
}

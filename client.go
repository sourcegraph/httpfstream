package httpfstream

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/garyburd/go-websocket/websocket"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Follow opens a WebSocket to the file at the given URL (which must be handled
// by httpfstream's HTTP handler) and returns the file's contents. The
// io.ReadCloser continues to return data (blocking as needed) if, and as long
// as, there is an active writer to the file.
func Follow(u *url.URL) (io.ReadCloser, error) {
	ws, resp, err := newClient(u, "FOLLOW")
	if err == websocket.ErrBadHandshake {
		err = errorFromResponse(resp, nil)
	}
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}

	return &webSocketReadCloser{ws}, nil
}

type webSocketReadCloser struct {
	ws *websocket.Conn
}

// Read implements io.Reader.
func (r *webSocketReadCloser) Read(p []byte) (n int, err error) {
	op, rdr, err := r.ws.NextReader()
	if err != nil {
		return 0, err
	}
	if op != websocket.OpText {
		return 0, errors.New("websocket op is not text")
	}
	n, err = rdr.Read(p)

	if err == io.EOF {
		return r.Read(p)
	}
	return
}

// Close implements io.Closer.
func (r *webSocketReadCloser) Close() error {
	return r.ws.Close()
}

// Append appends data from r to the file at the given URL.
func Append(u *url.URL, r io.Reader) error {
	w, err := OpenAppend(u)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}

// OpenAppend opens a WebSocket to the file at the given URL (which must point
// be handled by httpfstream's HTTP handler) and returns an io.WriteCloser that writes
// (via the WebSocket) to that file.
func OpenAppend(u *url.URL) (io.WriteCloser, error) {
	ws, resp, err := newClient(u, "APPEND")
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if err == websocket.ErrBadHandshake {
			err2 := errorFromResponse(resp, nil)
			if err2 != nil {
				return nil, err2
			}
			return nil, err
		}
		return nil, err
	}

	return &appendWriteCloser{new(bytes.Buffer), ws}, nil
}

type appendWriteCloser struct {
	io.Writer
	ws *websocket.Conn
}

// Write implements io.Writer.
func (pw *appendWriteCloser) Write(p []byte) (n int, err error) {
	pw.ws.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := pw.ws.NextWriter(websocket.OpText)
	if err != nil {
		return 0, err
	}
	defer w.Close()
	return w.Write(p)
}

// Write implements io.Closer.
func (pw *appendWriteCloser) Close() error {
	return pw.ws.Close()
}

func newClient(u *url.URL, method string) (*websocket.Conn, *http.Response, error) {
	var c net.Conn
	var err error
	hostport := hostPort(u)
	switch u.Scheme {
	case "http":
		c, err = net.Dial("tcp", hostport)
	case "https":
		c, err = tls.Dial("tcp", hostport, nil)
	default:
		return nil, nil, errors.New("unrecognized URL scheme")
	}
	if err != nil {
		return nil, nil, err
	}
	return websocket.NewClient(c, u, http.Header{xVerb: []string{method}}, readBufSize, writeBufSize)
}

func hostPort(u *url.URL) string {
	if strings.Contains(u.Host, ":") {
		return u.Host
	}
	return u.Host + ":" + u.Scheme
}

// errorFromResponse returns err if err != nil, or another non-nil error if resp
// indicates a non-HTTP 200 response.
func errorFromResponse(resp *http.Response, err error) error {
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return os.ErrNotExist
		default:
			return fmt.Errorf("HTTP status %d", resp.StatusCode)
		}
	}
	return nil
}

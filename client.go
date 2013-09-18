package httpfstream

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sqs/go-websocket/websocket"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

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

func (r *webSocketReadCloser) Close() error {
	return r.ws.Close()
}

func Append(u *url.URL, r io.Reader) error {
	w, err := OpenAppend(u)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}

func OpenAppend(u *url.URL) (io.WriteCloser, error) {
	ws, resp, err := newClient(u, "APPEND")
	if err != nil {
		if err == websocket.ErrBadHandshake {
			return nil, errorFromResponse(resp, nil)
		}
		return nil, err
	}

	return &appendWriteCloser{new(bytes.Buffer), ws}, nil
}

type appendWriteCloser struct {
	io.Writer
	ws *websocket.Conn
}

func (pw *appendWriteCloser) Write(p []byte) (n int, err error) {
	pw.ws.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := pw.ws.NextWriter(websocket.OpText)
	if err != nil {
		return 0, err
	}
	defer w.Close()
	return w.Write(p)
}

func (pw *appendWriteCloser) Close() error {
	return pw.ws.Close()
}

func newClient(u *url.URL, method string) (*websocket.Conn, *http.Response, error) {
	c, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, nil, err
	}
	return websocket.NewClient(c, u, http.Header{xVerb: []string{method}}, readBufSize, writeBufSize)
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

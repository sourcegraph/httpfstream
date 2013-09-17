package httpfstream

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/garyburd/go-websocket/websocket"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

func Get(u *url.URL) (io.ReadCloser, error) {
	req, err := http.NewRequest("HEAD", u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return nil, os.ErrNotExist
		default:
			return nil, fmt.Errorf("head failed with status %d", resp.StatusCode)
		}
	}

	ws, err := newClient(u, "GET")
	if err != nil {
		return nil, err
	}

	op, r, err := ws.NextReader()
	if err != nil {
		return nil, err
	}
	if op != websocket.OpText {
		return nil, errors.New("websocket op is not text")
	}

	return &webSocketReadCloser{r, ws}, nil
}

type webSocketReadCloser struct {
	io.Reader
	ws *websocket.Conn
}

func (r *webSocketReadCloser) Read(p []byte) (n int, err error) {
	return r.Reader.Read(p)
}

func (r *webSocketReadCloser) Close() error {
	return r.ws.Close()
}

func Put(u *url.URL, r io.Reader) error {
	w, err := OpenPut(u)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}

func OpenPut(u *url.URL) (io.WriteCloser, error) {
	req, err := http.NewRequest("HEAD", u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		switch resp.StatusCode {
		default:
			return nil, fmt.Errorf("head failed with status %d", resp.StatusCode)
		}
	}

	ws, err := newClient(u, "PUT")
	if err != nil {
		return nil, err
	}

	return &putWriteCloser{new(bytes.Buffer), ws}, nil
}

type putWriteCloser struct {
	io.Writer
	ws *websocket.Conn
}

func (pw *putWriteCloser) Write(p []byte) (n int, err error) {
	pw.ws.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := pw.ws.NextWriter(websocket.OpText)
	if err != nil {
		return 0, err
	}
	defer w.Close()
	return w.Write(p)
}

func (pw *putWriteCloser) Close() error {
	return pw.ws.Close()
}

func newClient(u *url.URL, method string) (*websocket.Conn, error) {
	c, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, err
	}
	ws, _, err := websocket.NewClient(c, u, http.Header{xMethod: []string{method}}, readBufSize, writeBufSize)
	return ws, err
}

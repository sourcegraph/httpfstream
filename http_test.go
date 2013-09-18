package httpfstream

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

type testServer struct {
	*httptest.Server
	dir string
}

func newTestServer() testServer {
	dir, err := ioutil.TempDir("", "httpfstream")
	if err != nil {
		panic("TempDir: " + err.Error())
	}
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		panic("MkdirAll: " + err.Error())
	}

	rootMux := http.NewServeMux()
	h := New(dir)
	h.Log = log.New(os.Stderr, "", 0)
	rootMux.Handle("/", h)
	return testServer{
		Server: httptest.NewServer(rootMux),
		dir:    dir,
	}
}

func (s testServer) close() {
	s.Server.Close()
	os.RemoveAll(s.dir)
}

func httpGET(t *testing.T, u *url.URL) string {
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("httpGET %s: %s", u, err)
	}
	defer resp.Body.Close()
	return string(readAll(t, resp.Body))
}

func readAll(t *testing.T, rdr io.Reader) []byte {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		t.Fatal("ReadAll", err)
	}
	return data
}

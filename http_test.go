package httpfstream

import (
	"github.com/sourcegraph/rwvfs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type testServer struct {
	*httptest.Server
	dir string
	fs  rwvfs.FileSystem
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
	fs := rwvfs.OS(dir)
	h := New(fs)
	h.Log = log.New(os.Stderr, "", 0)
	rootMux.Handle("/", h)
	return testServer{
		Server: httptest.NewServer(rootMux),
		dir:    dir,
		fs:     fs,
	}
}

func (s testServer) close() {
	s.Server.Close()
	os.RemoveAll(s.dir)
}

func readAll(t *testing.T, rdr io.Reader) []byte {
	if c, ok := rdr.(io.Closer); ok {
		defer c.Close()
	}
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		t.Fatal("ReadAll", err)
	}
	return data
}

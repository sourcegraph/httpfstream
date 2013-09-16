package httpfstream

import (
	"github.com/sourcegraph/rwvfs"
	"io"
	"io/ioutil"
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
	rootMux.Handle("/", New(fs))
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

func readAll(t *testing.T, rdr io.ReadCloser) []byte {
	defer rdr.Close()
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		t.Fatal("ReadAll", err)
	}
	return data
}

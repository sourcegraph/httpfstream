package httpfstream

import (
	"code.google.com/p/go.tools/godoc/vfs/httpfs"
	"github.com/sourcegraph/rwvfs"
	"log"
	"net/http"
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

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.FileServer(h.httpFS).ServeHTTP(w, r)
	case "PUT":
		h.Upload(w, r)
	default:
		http.Error(w, "only GET or PUT methods are supported", http.StatusBadRequest)
	}
}

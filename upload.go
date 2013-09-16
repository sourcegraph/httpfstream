package httpfstream

import (
	"io"
	"net/http"
	"os"
)

func (h handler) Upload(w http.ResponseWriter, r *http.Request) {
	if h.Log != nil {
		h.Log.Printf("PUT %s", r.URL.Path)
	}

	defer r.Body.Close()

	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		http.Error(w, "path must not end with '/'", http.StatusBadRequest)
		return
	}

	f, err := h.Root.OpenFile(r.URL.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL)
	if err != nil {
		http.Error(w, "failed to open destination file for writing: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, r.Body)
	if err != nil {
		http.Error(w, "failed to copy upload stream to file: "+err.Error(), http.StatusInternalServerError)
		return
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

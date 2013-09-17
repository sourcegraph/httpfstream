package httpfstream

import (
	"io"
	"log"
	"net/url"
	"testing"
	"time"
)

func waitForWrite() {
	time.Sleep(time.Millisecond * 50)
}

func TestStream(t *testing.T) {
	server := newTestServer()
	defer server.close()

	path := "/stream"
	u, _ := url.Parse(server.URL + path)

	log.SetFlags(0)

	w, err := OpenPut(u)
	if err != nil {
		t.Fatalf("OpenPut: %s", err)
	}

	r, err := Get(u)
	if err != nil {
		t.Fatalf("Get: %s", err)
	}

	data := []string{"foo", "bar", "baz", "qux"}
	var written string
	for _, d := range data {
		io.WriteString(w, d)
		written += d
		waitForWrite()

		// Check that PUT data is persisted.
		if got := getAll(t, u); written != got {
			t.Fatalf("want all == %q, got %q", written, got)
		}

		// Check that PUT data is sent to WebSocket reader.
		if got := string(readAll(t, r)); d != got {
			t.Errorf("want msg == %q, got %q", d, got)
		}
	}

	defer w.Close()
}

func getAll(t *testing.T, u *url.URL) string {
	r, err := Get(u)
	if err != nil {
		t.Fatalf("getAll %s: %s", u, err)
	}
	defer r.Close()
	return string(readAll(t, r))
}

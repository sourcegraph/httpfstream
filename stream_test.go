package httpfstream

import (
	"io"
	"log"
	"net/url"
	"strings"
	"testing"
	"time"
)

func waitForWrite() {
	time.Sleep(time.Millisecond * 10)
}

func TestStream(t *testing.T) {
	t.Parallel()
	server := newTestServer()
	defer server.close()

	path := "/stream"
	u, _ := url.Parse(server.URL + path)

	log.SetFlags(0)

	w, err := OpenAppend(u)
	if err != nil {
		t.Fatalf("OpenAppend: %s", err)
	}
	defer w.Close()

	r, err := Follow(u)
	if err != nil {
		t.Fatalf("Follow: %s", err)
	}
	defer r.Close()

	writedata := []string{"foo", "bar", "baz", "qux"}
	wantdata := []string{"abcfoo", "bar", "baz", "qux"}
	io.WriteString(w, "abc")
	for i, d := range writedata {
		io.WriteString(w, d)
		waitForWrite()

		// Check that APPEND data is persisted.
		want := strings.Join(wantdata[:i+1], "")
		if got := httpGET(t, u); want != got {
			t.Errorf("want all == %q, got %q", want, got)
			return
		}

		// Check that APPEND data is sent to WebSocket reader.
		followdata := string(limitRead(t, r, int64(len(wantdata[i]))))
		if wantdata[i] != followdata {
			t.Errorf("want msg == %q, got %q", wantdata[i], followdata)
		}
	}
}

func limitRead(t *testing.T, rdr io.Reader, n int64) []byte {
	lr := io.LimitReader(rdr, n)
	data := make([]byte, n)
	_, err := io.ReadAtLeast(lr, data, int(n))
	if err != nil {
		t.Fatal("limitRead", err)
	}
	return data
}

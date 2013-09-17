package httpfstream

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"testing"
	"time"
)

type getTest struct {
	path       string
	body       string
	writeFiles map[string]string
	err        error
}

func TestGet(t *testing.T) {
	tests := []getTest{
		{path: "/foo1", body: "bar", writeFiles: map[string]string{"/foo1": "bar"}},

		{path: "/doesntexist", err: os.ErrNotExist},
	}
	for _, test := range tests {
		testGet(t, test)
	}
}

func testGet(t *testing.T, test getTest) {
	label := test.path

	server := newTestServer()
	defer server.close()

	for path, data := range test.writeFiles {
		w, err := server.fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL)
		if err != nil {
			t.Fatalf("%s: fs.WriterOpen: %s", label, err)
		}
		_, err = w.Write([]byte(data))
		if err != nil {
			t.Fatalf("%s: Write: %s", label, err)
		}
		err = w.Close()
		if err != nil {
			t.Fatalf("%s: Close: %s", label, err)
		}
	}

	u, _ := url.Parse(server.URL + test.path)
	r, err := Get(u)
	if err == nil {
		defer r.Close()
	}
	if test.err != err {
		t.Errorf("%s: Get: want error %v, got %v", label, test.err, err)
		return
	}
	if test.err != nil {
		return
	}

	body := string(readAll(t, r))
	if test.body != body {
		t.Errorf("%s: want body == %q, got %q", label, test.body, body)
	}
}

type putTest struct {
	path     string
	data     io.Reader
	fileData string
	err      error
}

func TestPut(t *testing.T) {
	tests := []putTest{
		{path: "/foo1", data: bytes.NewReader([]byte("bar")), fileData: "bar"},
		{path: "/foo2", data: bytes.NewBuffer([]byte("bar")), fileData: "bar"},
		{path: "/foo3", data: &slowReader{R: &fixedReader{R: bytes.NewReader([]byte("quxx")), N: 2}, Wait: time.Millisecond * 25}, fileData: "quxx"},
	}
	for _, test := range tests {
		testPut(t, test)
	}
}

func testPut(t *testing.T, test putTest) {
	label := test.path

	server := newTestServer()
	defer server.close()

	u, _ := url.Parse(server.URL + test.path)
	err := Put(u, test.data)
	if test.err != err {
		t.Errorf("%s: Put: want error %v, got %v", label, test.err, err)
		return
	}
	if test.err != nil {
		return
	}

	time.Sleep(50 * time.Millisecond)

	_, err = server.fs.Stat(test.path)
	if err != nil {
		t.Errorf("%s: Stat: %s", label, err)
		return
	}

	f, err := server.fs.Open(test.path)
	if err != nil {
		t.Errorf("%s: Open: %s", label, err)
		return
	}
	fileData := string(readAll(t, f))
	if test.fileData != fileData {
		t.Errorf("%s: want fileData == %q, got %q", label, test.fileData, fileData)
	}
}

type slowReader struct {
	R         io.Reader
	Wait      time.Duration
	afterRead func(read []byte)
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	if r.afterRead != nil {
		defer func() {
			r.afterRead(p)
		}()
	}
	time.Sleep(r.Wait)
	return r.R.Read(p)
}

type fixedReader struct {
	R io.Reader
	N int
}

func (r *fixedReader) Read(p []byte) (n int, err error) {
	return io.LimitReader(r.R, int64(r.N)).Read(p)
}

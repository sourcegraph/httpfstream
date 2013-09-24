package httpfstream

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type followTest struct {
	path       string
	body       string
	writeFiles map[string]string
	err        error
}

func TestFollow(t *testing.T) {
	t.Parallel()

	tests := []followTest{
		{path: "/foo1", body: "bar", writeFiles: map[string]string{"/foo1": "bar"}},

		{path: "/doesntexist", err: os.ErrNotExist},
	}
	for _, test := range tests {
		testFollow(t, test)
	}
}

func testFollow(t *testing.T, test followTest) {
	label := test.path

	server := newTestServer()
	defer server.close()

	for path, data := range test.writeFiles {
		path = filepath.Join(server.dir, path)
		w, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0644)
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
	r, err := Follow(u)
	if err == nil {
		defer r.Close()
	}
	if test.err != err {
		t.Errorf("%s: Follow: want error %v, got %v", label, test.err, err)
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

type appendTest struct {
	path     string
	data     io.Reader
	fileData string
	err      error
}

func TestAppend(t *testing.T) {
	t.Parallel()

	tests := []appendTest{
		{path: "/foo1", data: bytes.NewReader([]byte("bar")), fileData: "bar"},
		{path: "/foo2", data: bytes.NewBuffer([]byte("bar")), fileData: "bar"},
		{path: "/foo3", data: &slowReader{R: &fixedReader{R: bytes.NewReader([]byte("quxx")), N: 2}, Wait: time.Millisecond * 10}, fileData: "quxx"},
	}
	for _, test := range tests {
		testAppend(t, test)
	}
}

func testAppend(t *testing.T, test appendTest) {
	label := test.path

	server := newTestServer()
	defer server.close()
	fpath := filepath.Join(server.dir, test.path)

	u, _ := url.Parse(server.URL + test.path)
	err := Append(u, test.data)
	if test.err != err {
		t.Errorf("%s: Append: want error %v, got %v", label, test.err, err)
		return
	}
	if test.err != nil {
		return
	}

	time.Sleep(10 * time.Millisecond)

	_, err = os.Stat(fpath)
	if err != nil {
		t.Errorf("%s: Stat: %s", label, err)
		return
	}

	f, err := os.Open(fpath)
	if err != nil {
		t.Errorf("%s: Open: %s", label, err)
		return
	}
	defer f.Close()
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

func TestHostPort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://example.com", "example.com:http"},
		{"https://example.com", "example.com:https"},
		{"http://example.com:1234", "example.com:1234"},
		{"https://example.com:1234", "example.com:1234"},
	}

	for _, test := range tests {
		u, err := url.Parse(test.input)
		if err != nil {
			t.Errorf("%s: url.Parse: %s", test.input, err)
			continue
		}
		got := hostPort(u)
		if test.expected != got {
			t.Errorf("%s: want %q, got %q", test.input, test.expected, got)
		}
	}
}

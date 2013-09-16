package httpfstream

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type uploadTest struct {
	path string
	data io.Reader

	// error responses
	statusCode int
	msg        string
}

func TestUpload(t *testing.T) {
	tests := []uploadTest{
		{path: "/foo", data: bytes.NewReader([]byte("bar")), statusCode: http.StatusOK},
		{path: "/foo", statusCode: http.StatusOK},

		{path: "/", statusCode: http.StatusBadRequest, msg: "path must not end with '/'"},
		{path: "/..", statusCode: http.StatusMovedPermanently},
		{path: "/../foo", statusCode: http.StatusMovedPermanently},
	}
	for _, test := range tests {
		testUpload(t, test)
	}
}

func testUpload(t *testing.T, test uploadTest) {
	label := test.path

	server := newTestServer()
	defer server.close()

	req, err := http.NewRequest("PUT", server.URL+test.path, nil)
	if err != nil {
		t.Fatalf("%s: NewRequest: %s", label, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s: Do: %s", label, err)
	}

	if test.statusCode != resp.StatusCode {
		t.Errorf("%s: want StatusCode == %d, got %d", label, test.statusCode, resp.StatusCode)
	}

	if test.statusCode >= 200 && test.statusCode <= 299 {
		_, err = server.fs.Stat(test.path)
		if err != nil {
			t.Errorf("%s: Stat: %s", label, err)
		}
	} else {
		msg := strings.TrimSpace(string(readAll(t, resp.Body)))
		if test.msg != msg {
			t.Errorf("%s: want error message %q, got %q", label, test.msg, msg)
		}
	}
}

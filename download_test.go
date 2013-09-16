package httpfstream

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

type downloadTest struct {
	path string
	body string

	writeFiles map[string]string

	// error responses
	statusCode int
	msg        string
}

func TestDownload(t *testing.T) {
	tests := []downloadTest{
		{path: "/foo", body: "bar", writeFiles: map[string]string{"/foo": "bar"}, statusCode: http.StatusOK},

		{path: "/doesntexist", statusCode: http.StatusNotFound, msg: "404 page not found"},
	}
	for _, test := range tests {
		testDownload(t, test)
	}
}

func testDownload(t *testing.T, test downloadTest) {
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

	req, err := http.NewRequest("GET", server.URL+test.path, nil)
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

	body := string(readAll(t, resp.Body))
	if test.statusCode >= 200 && test.statusCode <= 299 {
		if test.body != body {
			t.Errorf("%s: want data == %q, got %q", label, test.body, body)
		}
	} else {
		msg := strings.TrimSpace(body)
		if test.msg != msg {
			t.Errorf("%s: want error message %q, got %q", label, test.msg, msg)
		}
	}
}

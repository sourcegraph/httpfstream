package main

import (
	"flag"
	"fmt"
	"github.com/sourcegraph/httpfstream"
	"log"
	"net/http"
	"os"
)

var bindAddr = flag.String("http", ":8080", "HTTP bind address for server")
var root = flag.String("root", "/tmp/httpfstream", "storage root directory")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "httpfstream-server supports simultaneous, streaming file uploading and downloading over HTTP.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\thttpfstream-server [options]\n\n")
		fmt.Fprintf(os.Stderr, "The options are:\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Example usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tTo run on http://localhost:8080:\n")
		fmt.Fprintf(os.Stderr, "\t    $ httpfstream-server -http=:8080\n\n")
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
	}

	os.MkdirAll(*root, 0700)

	h := httpfstream.New(*root)
	h.Log = log.New(os.Stderr, "", 0)
	http.Handle("/", h)

	log.Printf("Starting server on %s\n", *bindAddr)
	err := http.ListenAndServe(*bindAddr, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}

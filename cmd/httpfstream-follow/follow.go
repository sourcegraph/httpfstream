package main

import (
	"flag"
	"fmt"
	"github.com/sourcegraph/httpfstream"
	"io"
	"log"
	"net/url"
	"os"
)

var verbose = flag.Bool("v", false, "show verbose output")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "httpfstream-follow fetches and follows data at a resource URL on an httpfstream server.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tcat FILE | httpfstream-follow [options] url\n\n")
		fmt.Fprintf(os.Stderr, "The options are:\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Example usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tTo follow data being written to http://localhost:8080/foo.txt by an httpfstream appender:\n")
		fmt.Fprintf(os.Stderr, "\t    $ httpfstream-follow http://localhost:8080/foo.txt\n\n")
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
	}

	log.SetFlags(0)

	urlstr := flag.Arg(0)
	u, err := url.Parse(urlstr)
	if err != nil {
		log.Fatalf("failed to parse URL %q: %s", urlstr, err)
	}

	if *verbose {
		log.Printf("following data at %s (ctrl-C to exit)", u)
	}

	r, err := httpfstream.Follow(u)
	if err != nil {
		log.Fatalf("failed to begin following %s: %s", u, err)
	}

	n, err := io.Copy(os.Stdout, r)
	if err != nil {
		log.Fatalf("error following %s: %s", u, err)
	}
	if *verbose {
		log.Printf("finished following (read %d bytes)", n)
	}
}

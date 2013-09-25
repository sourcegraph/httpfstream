package main

import (
	"flag"
	"fmt"
	"github.com/sourcegraph/httpfstream"
	"log"
	"net/url"
	"os"
)

var verbose = flag.Bool("v", false, "show verbose output")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "httpfstream-append streams data to a resource URL on an httpfstream server.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tcat FILE | httpfstream-append [options] url\n\n")
		fmt.Fprintf(os.Stderr, "The options are:\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Example usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tTo upload `foo.txt' to an httpfstream server at http://localhost:8080:\n")
		fmt.Fprintf(os.Stderr, "\t    $ cat foo.txt | httpfstream-append http://localhost:8080/foo.txt\n\n")
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
		log.Printf("appending from stdin to %s", u)
	}

	err = httpfstream.Append(u, os.Stdin)
	if err != nil {
		log.Fatalf("failed to append from stdin to %s: %s", u, err)
	}

	if *verbose {
		log.Printf("finished appending from stdin to %s", u)
	}
}

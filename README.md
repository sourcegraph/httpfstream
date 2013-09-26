httpfstream
==========

httpfstream provides HTTP handlers for simultaneous streaming uploads and
downloads of files, as well as persistence and a standalone server.

It allows a writer to `APPEND` data to a resource via a WebSocket and multiple
readers to `FOLLOW` updates to the resource using WebSockets.

Only one simultaneous appender is allowed for each resource. If there are no
appenders at an existing resource, the server returns the full data in an HTTP
200 (bypassing WebSockets) to a follower. If the resource has never been written
to, the server responds to a follower with HTTP 404.

* [Documentation on Sourcegraph](https://sourcegraph.com/github.com/sourcegraph/httpfstream)

[![Build Status](https://travis-ci.org/sourcegraph/httpfstream.png?branch=master)](https://travis-ci.org/sourcegraph/httpfstream)
[![xrefs](https://sourcegraph.com/api/repos/github.com/sourcegraph/httpfstream/badges/xrefs.png)](https://sourcegraph.com/github.com/sourcegraph/httpfstream)
[![funcs](https://sourcegraph.com/api/repos/github.com/sourcegraph/httpfstream/badges/funcs.png)](https://sourcegraph.com/github.com/sourcegraph/httpfstream)
[![top func](https://sourcegraph.com/api/repos/github.com/sourcegraph/httpfstream/badges/top-func.png)](https://sourcegraph.com/github.com/sourcegraph/httpfstream)
[![library users](https://sourcegraph.com/api/repos/github.com/sourcegraph/httpfstream/badges/library-users.png)](https://sourcegraph.com/github.com/sourcegraph/httpfstream)
[![status](https://sourcegraph.com/api/repos/github.com/sourcegraph/httpfstream/badges/status.png)](https://sourcegraph.com/github.com/sourcegraph/httpfstream)
[![Views in the last 24 hours](https://sourcegraph.com/api/repos/github.com/sourcegraph/httpfstream/counters/views-24h.png)](https://sourcegraph.com/github.com/sourcegraph/httpfstream)

Installation
------------

```bash
go get github.com/sourcegraph/httpfstream
```


Usage
-----

httpfstream supports 2 modes of usage: as a standalone server or as a Go
library.

### As a standalone server

The command `httpfstream-server` launches a server that allows clients to APPEND
and FOLLOW arbitrary file paths. Run with `-h` for more information.

For example, first install the commands:

```bash
$ go get github.com/sourcegraph/httpfstream/cmd/...
```

Then run the server with:

```bash
$ httpfstream-server -root=/tmp/httpfstream -http=:8080
```

Then launch a follower on `/foo.txt`:

```bash
$ httpfstream-follow -v http://localhost:8080/foo.txt
# keep this terminal window open
```

And start appending to `/foo.txt` in a separate terminal:

```bash
$ httpfstream-append -v http://localhost:8080/foo.txt
# start typing:
foo
bar
baz
# now exit: ctrl-C
```

Notice that the `httpfstream-follow` window echoes what you type into the
appender window. Once you close the appender, the follower quits as well.


### As a Go library

#### Server

The function [`httpfstream.New(root
string)`](https://sourcegraph.com/github.com/sourcegraph/httpfstream/symbols/go/github.com/sourcegraph/httpfstream/New)
takes the root file storage path as a parameter and returns an
[`http.Handler`](https://sourcegraph.com/code.google.com/p/go/symbols/go/code.google.com/p/go/src/pkg/net/http/Handler:type)
that lets clients `APPEND` and `FOLLOW` to paths it handles.

The file `cmd/httpfstream-server/server.go` contains a full example, summarized here:

```go
package main

import (
	"github.com/sourcegraph/httpfstream"
	"log"
	"net/http"
	"os"
)

func main() {
	h := httpfstream.New("/tmp/httpfstream")
	h.Log = log.New(os.Stderr, "", 0)
	http.Handle("/", h)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}
```

#### Appender

Clients can append data to a resource using either [`httpfstream.Append(u *url.URL,
r io.Reader) error`](https://sourcegraph.com/github.com/sourcegraph/httpfstream/symbols/go/github.com/sourcegraph/httpfstream/Append)
(if they already have an `io.Reader`) or [`httpfstream.OpenAppend(u *url.URL)
(io.WriteCloser,
error)`](https://sourcegraph.com/github.com/sourcegraph/httpfstream/symbols/go/github.com/sourcegraph/httpfstream/OpenAppend).

Click on the function names (linked above) to see full docs and usage examples
on Sourcegraph.


#### Follower

Clients can follow a resource's data using [`httpfstream.Follow(u *url.URL)
(io.ReadCloser, error)`](https://sourcegraph.com/github.com/sourcegraph/httpfstream/symbols/go/github.com/sourcegraph/httpfstream/Follow).

Click on the function names (linked above) to see full docs and usage examples
on Sourcegraph.


Contributing
------------

Patches and bug reports welcomed! Report issues and submit pull requests using
GitHub.

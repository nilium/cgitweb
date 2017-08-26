// What follows is code borrowed from the stdlib net/http/cgi package (and slightly modified).
// Its license can be found under the COPYING file or, if not provided, at <https://golang.org/LICENSE>.

package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// serve executes the provided Handler on the given HTTP request.
// If there's no current CGI environment an error is returned.
// The provided handler may be nil to use http.DefaultServeMux.
func serve(req *http.Request, handler http.Handler) error {
	if handler == nil {
		handler = http.DefaultServeMux
	}
	rw := &response{
		req:    req,
		header: make(http.Header),
		bufw:   bufio.NewWriter(os.Stdout),
	}
	handler.ServeHTTP(rw, req)
	rw.Write(nil) // make sure a response is sent
	if err := rw.bufw.Flush(); err != nil {
		return err
	}
	return nil
}

func envMap(env []string) map[string]string {
	m := make(map[string]string)
	for _, kv := range env {
		if idx := strings.Index(kv, "="); idx != -1 {
			m[kv[:idx]] = kv[idx+1:]
		}
	}
	return m
}

type response struct {
	req        *http.Request
	header     http.Header
	bufw       *bufio.Writer
	headerSent bool
}

func (r *response) Flush() {
	r.bufw.Flush()
}

func (r *response) Header() http.Header {
	return r.header
}

func (r *response) Write(p []byte) (n int, err error) {
	if !r.headerSent {
		r.WriteHeader(http.StatusOK)
	}
	return r.bufw.Write(p)
}

func (r *response) WriteHeader(code int) {
	if r.headerSent {
		logger.Printf("CGI attempted to write header twice on request for %s", r.req.URL)
		return
	}
	r.headerSent = true
	fmt.Fprintf(r.bufw, "Status: %d %s\r\n", code, http.StatusText(code))

	// Set a default Content-Type
	if _, hasType := r.header["Content-Type"]; !hasType {
		r.header.Add("Content-Type", "text/html; charset=utf-8")
	}

	r.header.Write(r.bufw)
	r.bufw.WriteString("\r\n")
	r.bufw.Flush()
}

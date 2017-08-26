// Command cgitweb is a dumb gitweb-to-cgit proxy specifically for handling gitweb links in gerrit.
// In practice, this doesn't work very well because gerrit's handling of gitweb URLs is basically
// hard-coded to a useless value.
//
// This doesn't handle all gitweb requests or actions or parameters, only those immediately used by
// gerrit.
//
// This package contains code borrowed from net/http/cgi and modified to allow a few odd cases
// through.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"net/url"
	"os"
	"path"
	"strings"
)

var logger = log.New(os.Stderr, "", log.LUTC|log.Lmicroseconds|log.Ltime)

func usage() {
	fmt.Println("Usage: cgitweb.cgi [CGITWEBRC_PATH]")
	os.Exit(2)
}

func main() {
	flag.CommandLine.Usage = usage
	flag.Parse()

	// load config
	cgitwebConfPath := "/usr/local/etc/cgitwebrc"
	if argv := flag.Args(); len(argv) == 1 {
		cgitwebConfPath = argv[0]
	} else if len(argv) != 0 {
		usage()
	}

	if confData, err := ioutil.ReadFile(cgitwebConfPath); err == nil {
		config.load(confData)
	} else if os.IsNotExist(err) {
		// nop
	} else {
		log.Fatal("error reading cgitwebrc: ", err)
	}

	// Check cgit.cgi path
	cgitCGI := config.getString("cgit-path", false, "/usr/local/www/cgit/cgit.cgi")
	switch stat, err := os.Stat(cgitCGI); {
	case err != nil:
		logger.Fatal("cannot stat cgit executable: ", err)
	case stat.Mode()&0111 == 0:
		logger.Fatal(cgitCGI, ": cgit is not executable")
	}

	// Build request environment
	env := envMap(os.Environ())
	if cenv := config.getPrefix("env.", true).toEnvironment(); len(cenv) > 0 {
		for k, v := range envMap(cenv) {
			env[k] = v
		}
	}

	req, err := cgi.RequestFromMap(env)
	if err != nil {
		logger.Fatal("error parsing request: ", err)
	}

	if maxSize := config.getInt64("max-request-size", -1); maxSize >= 0 && req.ContentLength > maxSize {
		logger.Fatal("request exceeds maximum-request-size=", maxSize, ": ", req.ContentLength)
	}

	// Restrict content length
	if size := req.ContentLength; size > 0 {
		req.Body = ioutil.NopCloser(io.LimitReader(os.Stdin, size))
	} else {
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte{}))
	}

	cgitHandler := cgi.Handler{
		Path:   cgitCGI,
		Args:   config.getStrings("cgit-arg"),
		Root:   config.getString("prefix", true, "/"),
		Env:    config.getPrefix("cgit-env.", true).toEnvironment(),
		Logger: logger,
	}

	// remap gitweb parameters to cgit -- this only covers query parameters, it does not support
	// gitweb path parameters right now. This is mainly because this is written to get around
	// gerrit being super-broken if you don't either use gitiles or gitweb, so it maps gitweb to
	// cgit so that gerrit can pretend that it's using gitweb.
	remap(req)
	serve(req, &cgitHandler)
}

func remap(req *http.Request) {
	var (
		u       = req.URL
		form    = u.Query()
		project = form.Get("p")
		action  = form.Get("a")
		f       = form.Get("f")
		h       = form.Get("h")
		hb      = form.Get("hb")
	)

	if config.getBool("trim-suffix", false) {
		trimmed := strings.TrimSuffix(project, ".git")
		if !strings.HasSuffix(trimmed, "/") {
			project = trimmed
		}
	}

	switch action {
	case "summary":
		u.Path = path.Join(project, "summary")
	case "commit":
		u.Path = path.Join(project, "commit/")
		u.RawQuery = url.Values{"id": {h}}.Encode()
	case "shortlog", "history":
		u.Path = path.Join(project, "log/", f)
		u.RawQuery = url.Values{"h": {h}}.Encode()
	case "tag":
		u.Path = path.Join(project, "tag/")
		u.RawQuery = url.Values{"h": {h}}.Encode()
	case "tree", "":
		u.Path = path.Join(project, "tree/", f)
		u.RawQuery = url.Values{"h": {hb}}.Encode()
	}
}

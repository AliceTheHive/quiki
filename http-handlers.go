// Copyright (c) 2017, Mitchell Cooper
package main

import (
	"bytes"
	wikiclient "github.com/cooper/go-wikiclient"
	"net/http"
	"regexp"
	"strconv"
)

// master handler
func handleRoot(w http.ResponseWriter, r *http.Request) {
	var delayedWiki wikiInfo

	// try each wiki
	for _, wiki := range wikis {

		// wrong root
		wikiRoot := wiki.conf.Get("root.wiki")
		if r.URL.Path != wikiRoot && r.URL.Path != wikiRoot+"/" {
			continue
		}

		// no main page configured
		mainPage := wiki.conf.Get("main_page")
		if mainPage == "" {
			continue
		}

		// wrong host
		if wiki.host != r.Host {

			// if the wiki host is empty, it is the fallback wiki.
			// delay it until we've checked all other wikis.
			if wiki.host == "" && delayedWiki.name == "" {
				delayedWiki = wiki
			}

			continue
		}

		// host matches
		delayedWiki = wiki
		break
	}

	// try the delayed wiki
	if delayedWiki.name != "" {
		http.Redirect(
			w, r,
			delayedWiki.conf.Get("root.page")+
				"/"+delayedWiki.conf.Get("main_page"),
			http.StatusMovedPermanently,
		)
		return
	}

	// anything else is a 404
	http.NotFound(w, r)
}

// page request
func handlePage(wiki wikiInfo, relPath string, w http.ResponseWriter, r *http.Request) {
	res, err := wiki.client.DisplayPage(relPath)

	// wikiclient error or 'not found' response
	if handleError(wiki, err, w, r) || handleError(wiki, res, w, r) {
		return
	}

	// other response
	switch res.Get("type") {

	// page redirect
	case "redirect":
		http.Redirect(w, r, wiki.conf.Get("root.page")+"/"+res.Get("name"), http.StatusMovedPermanently)

	// page content
	case "page":
		renderTemplate(wiki, w, "page", wikiPage{
			Res:        res,
			Title:      res.Get("title"),
			WikiTitle:  wiki.title,
			WikiLogo:   wiki.template.logo,
			WikiRoot:   wiki.conf.Get("root.wiki"),
			StaticRoot: wiki.template.staticRoot,
			navigation: wiki.conf.GetSlice("navigation"),
		})

	// anything else
	default:
		http.NotFound(w, r)
	}
}

// image request
func handleImage(wiki wikiInfo, relPath string, w http.ResponseWriter, r *http.Request) {
	res, err := wiki.client.DisplayImage(relPath, 0, 0)
	if handleError(wiki, err, w, r) || handleError(wiki, res, w, r) {
		return
	}
	http.ServeFile(w, r, res.Get("path"))
}

// this is set true when calling handlePage for the error page. this way, if an
// error occurs when trying to display the error page, we don't infinitely loop
// between handleError and handlePage
var useLowLevelError bool

func handleError(wiki wikiInfo, errMaybe interface{}, w http.ResponseWriter, r *http.Request) bool {
	msg := "An unknown error has occurred"
	switch err := errMaybe.(type) {

	// if there's no error, stop
	case nil:
		return false

	// message of type "not found" is an error; otherwise, stop
	case wikiclient.Message:
		if err.Get("type") != "not found" {
			return false
		}
		msg = err.Get("error")

		// string
	case string:
		msg = err

		// error
	case error:
		msg = err.Error()

	}

	// if we have an error page, use it
	errorPage := wiki.conf.Get("error_page")
	if !useLowLevelError && errorPage != "" {
		useLowLevelError = true
		handlePage(wiki, errorPage, w, r)
		useLowLevelError = false
		return true
	}

	// generic error response
	http.Error(w, msg, http.StatusNotFound)
	return true
}

func renderTemplate(wiki wikiInfo, w http.ResponseWriter, templateName string, p wikiPage) {
	var buf bytes.Buffer
	err := wiki.template.template.ExecuteTemplate(&buf, templateName+".tpl", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", strconv.FormatInt(int64(buf.Len()), 10))
	w.Write(buf.Bytes())
}

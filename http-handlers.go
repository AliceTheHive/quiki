// Copyright (c) 2017, Mitchell Cooper
package main

import (
	"fmt"
	"github.com/cooper/quiki/wikiclient"
	"log"
	"net/http"
)

func handlePage(c wikiclient.Client, relPath string, w http.ResponseWriter, r *http.Request) {
	res, err := c.Request(wikiclient.NewMessage("page", map[string]interface{}{
		"name": relPath,
	}))
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(res)
}

func handleImage(c wikiclient.Client, relPath string, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, relPath, c, r)
}

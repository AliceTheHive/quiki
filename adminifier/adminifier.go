// Package adminifier provides an administrative panel with a web-based editor.
package adminifier

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cooper/quiki/webserver"
	_ "github.com/cooper/quiki/webserver" // access existing ServeMux and config
	"github.com/cooper/quiki/wikifier"
	"github.com/pkg/errors"
)

var tmpl *template.Template
var mux *http.ServeMux
var conf *wikifier.Page
var host, root, dirAdminifier string

// Configure sets up adminifier on webserver.ServeMux using webserver.Conf.
func Configure() {
	conf = webserver.Conf
	mux = webserver.Mux

	// do nothing if not enabled
	if enable, _ := conf.GetBool("adminifier.enable"); !enable {
		return
	}

	// extract strings
	for key, ptr := range map[string]*string{
		"server.dir.adminifier": &dirAdminifier,
		"adminifier.host":       &host,
		"adminifier.root":       &root,
	} {
		str, err := conf.GetStr(key)
		if err != nil {
			log.Fatal(err)
		}
		*ptr = str
	}

	dirAdminifier = filepath.FromSlash(dirAdminifier)
	root += "/"

	// setup static files server
	if err := setupStatic(filepath.Join(dirAdminifier, "adminifier-static")); err != nil {
		log.Fatal(errors.Wrap(err, "setup adminifier-static"))
	}

	// create template
	tmpl = template.Must(tmpl.ParseGlob(filepath.Join(dirAdminifier, "template", "*.tpl")))

	// main handler
	mux.HandleFunc(host+root, handleRoot)
	log.Println("registered adminifier root: " + host + root)

	// template handlers
	for _, tmplName := range tmplHandlers {
		mux.HandleFunc(host+root+tmplName, handleTemplate)
	}

	// function handlers
	for name, function := range funcHandlers {
		mux.HandleFunc(host+root+name, function)
	}

	// TODO: handlers for each site at shortcode/
}

func setupStatic(staticPath string) error {
	staticRoot := root + "adminifier-static/"
	if stat, err := os.Stat(staticPath); err != nil || !stat.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return err
	}
	fileServer := http.FileServer(http.Dir(staticPath))
	mux.Handle(host+staticRoot, http.StripPrefix(staticRoot, fileServer))
	return nil
}
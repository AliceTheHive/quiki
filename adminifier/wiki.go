package adminifier

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cooper/quiki/authenticator"
	"github.com/cooper/quiki/webserver"
	"github.com/cooper/quiki/wiki"
	"github.com/pkg/errors"
)

var javascriptTemplates string

var frameHandlers = map[string]func(*wikiRequest){
	"dashboard":     handleDashboardFrame,
	"pages":         handlePagesFrame,
	"categories":    handleCategoriesFrame,
	"images":        handleImagesFrame,
	"models":        handleModelsFrame,
	"settings":      handleSettingsFrame,
	"edit-page":     handleEditPageFrame,
	"edit-model":    handleEditModelFrame,
	"switch-branch": handleSwitchBranchFrame,
}

var wikiFuncHandlers = map[string]func(*wikiRequest){
	"switch-branch/": handleSwitchBranch,
	"create-branch":  handleCreateBranch,
}

// wikiTemplate members are available to all wiki templates
type wikiTemplate struct {
	User      *authenticator.User // user
	Shortcode string              // wiki shortcode
	WikiTitle string              // wiki title
	Branch    string              // selected branch
	Static    string              // adminifier-static root
	AdminRoot string              // adminifier root
	Root      string              // wiki root
}

type wikiRequest struct {
	shortcode string
	wikiRoot  string
	wi        *webserver.WikiInfo
	w         http.ResponseWriter
	r         *http.Request
	tmplName  string
	dot       interface{}
	err       error
}

// TODO: verify session on ALL wiki handlers

func setupWikiHandlers(shortcode string, wi *webserver.WikiInfo) {

	// each of these URLs generates wiki.tpl
	for _, which := range []string{
		"dashboard", "pages", "categories",
		"images", "models", "settings", "help",
		"edit-page", "edit-model", "switch-branch",
	} {
		mux.HandleFunc(host+root+shortcode+"/"+which, func(w http.ResponseWriter, r *http.Request) {
			handleWiki(shortcode, wi, w, r)
		})
	}

	// frames to load via ajax
	frameRoot := root + shortcode + "/frame/"
	mux.HandleFunc(host+frameRoot, func(w http.ResponseWriter, r *http.Request) {

		// check logged in
		if !sessMgr.GetBool(r.Context(), "loggedIn") {
			http.Redirect(w, r, root+"login", http.StatusTemporaryRedirect)
			return
		}

		frameName := strings.TrimPrefix(r.URL.Path, frameRoot)
		tmplName := "frame-" + frameName + ".tpl"

		// call func to create template params
		var dot interface{} = nil
		if handler, exist := frameHandlers[frameName]; exist {

			// possibly switch wiki branches
			userWiki := wi
			branchName := sessMgr.GetString(r.Context(), "branch")
			if branchName != "" {
				branchWiki, err := wi.Branch(branchName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				userWiki = wi.Copy(branchWiki)
			}

			// create wiki request
			wr := &wikiRequest{
				shortcode: shortcode,
				wikiRoot:  root + shortcode,
				wi:        userWiki,
				w:         w,
				r:         r,
			}
			dot = wr

			// call handler
			handler(wr)

			// handler returned an error
			if wr.err != nil {
				panic(wr.err)
			}

			// handler was successful
			if wr.dot != nil {
				dot = wr.dot
			}
			if wr.tmplName != "" {
				tmplName = wr.tmplName
			}
		}

		// frame template does not exist
		if exist := tmpl.Lookup(tmplName); exist == nil {
			http.NotFound(w, r)
			return
		}

		// execute frame template with dot
		err := tmpl.ExecuteTemplate(w, tmplName, dot)

		// error occurred in template execution
		if err != nil {
			panic(err)
		}
	})

	// functions
	funcRoot := root + shortcode + "/func/"
	for funcName, thisHandler := range wikiFuncHandlers {
		handler := thisHandler
		mux.HandleFunc(host+funcRoot+funcName, func(w http.ResponseWriter, r *http.Request) {

			// check logged in
			//
			// TODO: everything in func/ will be JSON,
			// so return a "not logged in" error to present login popup
			// rather than redirecting
			//
			if !sessMgr.GetBool(r.Context(), "loggedIn") {
				http.Redirect(w, r, root+"login", http.StatusTemporaryRedirect)
				return
			}

			// possibly switch wiki branches
			userWiki := wi
			branchName := sessMgr.GetString(r.Context(), "branch")
			if branchName != "" {
				branchWiki, err := wi.Branch(branchName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				userWiki = wi.Copy(branchWiki)
			}

			// create wiki request
			wr := &wikiRequest{
				shortcode: shortcode,
				wikiRoot:  root + shortcode,
				wi:        userWiki,
				w:         w,
				r:         r,
			}

			// call handler
			handler(wr)

			// handler returned an error
			if wr.err != nil {
				panic(wr.err)
			}
		})
	}
}

func handleWiki(shortcode string, wi *webserver.WikiInfo, w http.ResponseWriter, r *http.Request) {

	// check logged in
	if !sessMgr.GetBool(r.Context(), "loggedIn") {
		http.Redirect(w, r, root+"login", http.StatusTemporaryRedirect)
		return
	}

	// load javascript templates
	if javascriptTemplates == "" {
		files, _ := filepath.Glob(dirAdminifier + "/template/js-tmpl/*.tpl")
		for _, fileName := range files {
			data, _ := ioutil.ReadFile(fileName)
			javascriptTemplates += string(data)
		}
	}

	err := tmpl.ExecuteTemplate(w, "wiki.tpl", struct {
		JSTemplates template.HTML
		wikiTemplate
	}{
		template.HTML(javascriptTemplates),
		getGenericTemplate(&wikiRequest{
			shortcode: shortcode,
			wi:        wi,
			w:         w,
			r:         r,
		}),
	})
	if err != nil {
		panic(err)
	}
}

func handleDashboardFrame(wr *wikiRequest) {
}

func handlePagesFrame(wr *wikiRequest) {
	handleFileFrames(wr, wr.wi.Pages())
}

func handleImagesFrame(wr *wikiRequest) {
	handleFileFrames(wr, wr.wi.Images(), "d")
}

func handleModelsFrame(wr *wikiRequest) {
	handleFileFrames(wr, wr.wi.Models())
}

func handleCategoriesFrame(wr *wikiRequest) {
	handleFileFrames(wr, wr.wi.Categories())
}

func handleFileFrames(wr *wikiRequest, results interface{}, extras ...string) {
	res, err := json.Marshal(map[string]interface{}{
		"sort_types": append([]string{"a", "c", "u", "m"}, extras...),
		"results":    results,
	})
	if err != nil {
		wr.err = err
		return
	}
	wr.dot = struct {
		JSON  template.HTML
		Order string
		wikiTemplate
	}{
		JSON:         template.HTML("<!--JSON\n" + string(res) + "\n-->"),
		Order:        "m-",
		wikiTemplate: getGenericTemplate(wr),
	}
}

func handleSettingsFrame(wr *wikiRequest) {
	// serve editor for the config file
	handleEditor(wr, wr.wi.ConfigFile, "wiki.conf", "Configuration file", false, true)
}

func handleEditPageFrame(wr *wikiRequest) {
	q := wr.r.URL.Query()

	// no page filename provided
	name := q.Get("page")
	if name == "" {
		wr.err = errors.New("no page filename provided")
		return
	}

	// find the page. if File is empty, it doesn't exist
	info := wr.wi.PageInfo(name)
	if info.File == "" {
		wr.err = errors.New("page does not exist")
		return
	}

	// serve editor
	handleEditor(wr, info.Path, info.File, info.Title, false, false)
}

func handleEditModelFrame(wr *wikiRequest) {
	q := wr.r.URL.Query()

	// no page filename provided
	name := q.Get("page")
	if name == "" {
		wr.err = errors.New("no model filename provided")
		return
	}

	// find the model. if File is empty, it doesn't exist
	info := wr.wi.ModelInfo(name)
	if info.File == "" {
		wr.err = errors.New("model does not exist")
		return
	}

	// serve editor
	handleEditor(wr, info.Path, info.File, info.File, true, false)
}

func handleEditor(wr *wikiRequest, path, file, title string, model, config bool) {
	wr.tmplName = "frame-editor.tpl"

	// call DisplayFile to get the content
	var fileRes wiki.DisplayFile
	switch r := wr.wi.DisplayFile(path).(type) {
	case wiki.DisplayFile:
		fileRes = r
	case wiki.DisplayError:
		wr.err = errors.New(r.DetailedError)
		return
	default:
		wr.err = errors.New("unknown error occurred in DisplayFile")
		return
	}

	wr.dot = struct {
		Found   bool
		JSON    template.HTML
		Model   bool   // true if editing a model
		Config  bool   // true if editing config
		Title   string // page title or filename
		File    string // filename
		Content string // file content
		wikiTemplate
	}{
		Found:        true,
		JSON:         template.HTML("<!--JSON\n{}\n-->"), // TODO
		Model:        model,
		Config:       config,
		Title:        title,
		File:         file,
		Content:      fileRes.Content,
		wikiTemplate: getGenericTemplate(wr),
	}
}

func handleSwitchBranchFrame(wr *wikiRequest) {
	branches, err := wr.wi.BranchNames()
	if err != nil {
		wr.err = err
		return
	}
	wr.dot = struct {
		Branches []string
		wikiTemplate
	}{
		Branches:     branches,
		wikiTemplate: getGenericTemplate(wr),
	}
}

func handleSwitchBranch(wr *wikiRequest) {
	relPath := strings.TrimPrefix(wr.r.URL.Path, wr.wikiRoot+"/func/switch-branch/")
	if relPath == "" {
		wr.err = errors.New("no branch selected")
		return
	}

	// fetch the branch
	_, wr.err = wr.wi.Branch((relPath))
	if wr.err != nil {
		return
	}

	// set branch
	sessMgr.Put(wr.r.Context(), "branch", relPath)

	// TODO: when this request is submitted by JS, the UI can just reload
	// the current frame so the user stays on the same page, just in new branch

	// redirect back to dashboard
	http.Redirect(wr.w, wr.r, wr.wikiRoot+"/dashboard", http.StatusTemporaryRedirect)
}

func handleCreateBranch(wr *wikiRequest) {

	// missing parameters or malformed request
	// TODO: need a different version of parsePost that returns JSON errors
	if !parsePost(wr.w, wr.r, "branch") {
		return
	}

	// switch branches
	branchName := wr.r.Form.Get("branch")
	_, err := wr.wi.NewBranch(branchName)
	if err != nil {
		wr.err = err
		return
	}
	sessMgr.Put(wr.r.Context(), "branch", branchName)

	// redirect back to dashboard
	http.Redirect(wr.w, wr.r, wr.wikiRoot+"/dashboard", http.StatusTemporaryRedirect)
}

func getGenericTemplate(wr *wikiRequest) wikiTemplate {
	return wikiTemplate{
		User:      sessMgr.Get(wr.r.Context(), "user").(*authenticator.User),
		Branch:    sessMgr.GetString(wr.r.Context(), "branch"),
		Shortcode: wr.shortcode,
		WikiTitle: wr.wi.Title,
		AdminRoot: strings.TrimRight(root, "/"),
		Static:    root + "adminifier-static",
		Root:      root + wr.shortcode,
	}
}

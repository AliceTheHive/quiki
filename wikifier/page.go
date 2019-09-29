package wikifier

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	httpdate "github.com/Songmu/go-httpdate"
	strip "github.com/grokify/html-strip-tags-go"
)

// Page represents a single page or article, generally associated with a .page file.
// It provides the most basic public interface to parsing with the wikifier engine.
type Page struct {
	Source   string  // source content
	FilePath string  // Path to the .page file
	VarsOnly bool    // True if Parse() should only extract variables
	Opt      PageOpt // page options
	styles   []styleEntry
	parser   *parser // wikifier parser instance
	main     block   // main block
	images   map[string][][]int
	*variableScope
}

// NewPage creates a page given its filepath.
func NewPage(filePath string) *Page {
	return &Page{FilePath: filePath, Opt: defaultPageOpt, variableScope: newVariableScope()}
}

// NewPageSource creates a page given some source code.
func NewPageSource(source string) *Page {
	return &Page{Source: source, Opt: defaultPageOpt, variableScope: newVariableScope()}
}

// Parse opens the page file and attempts to parse it, returning any errors encountered.
func (p *Page) Parse() error {
	p.parser = newParser()
	p.main = p.parser.block

	// create reader from file path or source code provided
	var reader io.Reader
	if p.Source != "" {
		reader = strings.NewReader(p.Source)
	} else if p.FilePath != "" {
		file, err := os.Open(p.FilePath)
		if err != nil {
			return err
		}
		defer file.Close()
		reader = file
	} else {
		return errors.New("neither Source nor FilePath provided")
	}

	// parse line-by-line
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if err := p.parser.parseLine(scanner.Bytes(), p); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// TODO: check if p.parser.catch != main block

	// parse the blocks, unless we only want vars
	if !p.VarsOnly {
		p.main.parse(p)
	}

	// inject variables set in the page to page opts
	if err := InjectPageOpt(p, &p.Opt); err != nil {
		// TODO: position
		return err
	}

	return nil
}

// HTML generates and returns the HTML code for the page.
// The page must be parsed with Parse before attempting this method.
func (p *Page) HTML() HTML {
	// TODO: cache and then recursively destroy elements
	return generateBlock(p.main, p)
}

// Exists is true if the page exists.
func (p *Page) Exists() bool {
	if p.Source != "" {
		return true
	}
	_, err := os.Stat(p.FilePath)
	return err == nil
}

// Name returns the resolved page name, with or without extension.
//
// This does NOT take symbolic links into account.
// It DOES include the page prefix, however, if applicable.
//
func (p *Page) Name() string {
	dir := p.Opt.Dir.Page
	path := p.Path()
	name := strings.TrimPrefix(path, dir)
	name = strings.TrimPrefix(name, "/")
	if strings.Index(path, dir) != -1 {
		return filepath.Base(p.RelPath())
	}
	return name
}

// NameNE returns the resolved page name with No Extension.
func (p *Page) NameNE() string {
	return PageNameNE(p.Name())
}

// Prefix returns the page prefix.
//
// For example, for a page named a/b.page, this is a.
// For a page named a.page, this is an empty string.
//
func (p *Page) Prefix() string {
	dir := strings.TrimSuffix(filepath.Dir(p.Name()), "/")
	if dir == "." {
		return ""
	}
	return dir
}

// Path returns the absolute path to the page as resolved.
// If the path does not resolve, returns an empty string.
func (p *Page) Path() string {
	path, _ := filepath.Abs(p.RelPath())
	return path
}

// RelName returns the unresolved page filename, with or without extension.
// This does NOT take symbolic links into account.
// It is not guaranteed to exist.
func (p *Page) RelName() string {
	dir := p.Opt.Dir.Page
	path := p.RelPath()
	name := strings.TrimPrefix(path, dir)
	name = strings.TrimPrefix(name, "/")
	if strings.Index(path, dir) != -1 {
		return filepath.Base(p.RelPath())
	}
	return name
}

// RelNameNE returns the unresolved page name with No Extension, relative to
// the page directory option.
// This does NOT take symbolic links into account.
// It is not guaranteed to exist.
func (p *Page) RelNameNE() string {
	return PageNameNE(p.RelName())
}

// RelPath returns the unresolved file path to the page.
// It may be a relative or absolute path.
// It is not guaranteed to exist.
func (p *Page) RelPath() string {
	return p.FilePath
}

// Redirect returns the location to which the page redirects, if any.
// This may be a relative or absolute URL, suitable for use in a Location header.
func (p *Page) Redirect() string {

	// symbolic link redirect
	if p.IsSymlink() {
		return p.Opt.Root.Page + "/" + p.NameNE()
	}

	// @page.redirect
	if link, err := p.GetStr("page.redirect"); err != nil {
		// FIXME: is there anyway to produce a warning for wrong variable type?
	} else if ok, _, target, _, _, _ := parseLink(link); ok {
		return target
	}

	return ""
}

// IsSymlink returns true if the page is a symbolic link to another file within
// the page directory. If it is symlinked to somewhere outside the page directory,
// it is treated as a normal page rather than a redirect.
func (p *Page) IsSymlink() bool {
	if !strings.HasPrefix(p.Prefix(), p.Opt.Dir.Page) {
		return false
	}
	fi, _ := os.Lstat(p.RelPath())
	return fi.Mode()&os.ModeSymlink != 0
}

// Created returns the page creation time.
func (p *Page) Created() time.Time {
	var t time.Time
	// FIXME: maybe produce a warning if this is not in the right format
	created, _ := p.GetStr("page.created")
	if created == "" {
		return t
	}
	if unix, err := strconv.ParseInt(created, 10, 0); err != nil {
		return time.Unix(unix, 0)
	}
	t, _ = httpdate.Str2Time(created, time.UTC)
	return t
}

// Modified returns the page modification time.
func (p *Page) Modified() time.Time {
	fi, _ := os.Lstat(p.Path())
	return fi.ModTime()
}

// CachePath returns the absolute path to the page cache file.
func (p *Page) CachePath() string {
	// FIXME: makedir
	path, _ := filepath.Abs(p.Opt.Dir.Cache + "/page/" + p.Name() + ".cache")
	return path
}

// CacheModified returns the page cache file time.
func (p *Page) CacheModified() time.Time {
	fi, _ := os.Lstat(p.CachePath())
	return fi.ModTime()
}

// SearchPath returns the absolute path to the page search text file.
func (p *Page) SearchPath() string {
	// FIXME: makedir
	path, _ := filepath.Abs(p.Opt.Dir.Cache + "/page/" + p.Name() + ".txt")
	return path
}

// # page info to be used in results, stored in cats/cache files
// sub page_info {
//     my $page = shift;
//     return filter_nonempty {
//         mod_unix    => $page->modified,
//         created     => $page->created,
//         draft       => $page->draft,
//         generated   => $page->generated,
//         redirect    => $page->redirect,
//         fmt_title   => $page->fmt_title,
//         title       => $page->title,
//         author      => $page->author
//     };
// }

// Draft returns true if the page is marked as a draft.
func (p *Page) Draft() bool {
	b, _ := p.GetBool("page.draft")
	return b
}

// Generated returns true if the page was auto-generated
// from some other source content.
func (p *Page) Generated() bool {
	b, _ := p.GetBool("page.generated")
	return b
}

// Author returns the page author's name, if any.
func (p *Page) Author() string {
	s, _ := p.GetStr("page.author")
	return s
}

// FmtTitle returns the page title, preserving any possible text formatting.
func (p *Page) FmtTitle() HTML {
	s, _ := p.GetStr("page.title")
	return HTML(s)
}

// Title returns the page title with HTML text formatting tags stripped.
func (p *Page) Title() string {
	return strip.StripTags(string(p.FmtTitle()))
}

// TitleOrName returns the result of Title if available, otherwise that of Name.
func (p *Page) TitleOrName() string {
	if title := p.Title(); title != "" {
		return title
	}
	return p.Name()
}

// resets the parser
func (p *Page) resetParseState() {
	// TODO: recursively destroy blocks
	p.parser = nil
}

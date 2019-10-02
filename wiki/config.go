package wiki

import (
	"fmt"

	"github.com/cooper/quiki/wikifier"
	"github.com/pkg/errors"
)

var defaultWikiOpt = wikifier.PageOpt{
	Page: wikifier.PageOptPage{
		EnableTitle: true,
		EnableCache: true,
	},
	Dir: wikifier.PageOptDir{
		Wikifier: ".",
		Wiki:     "",
		Image:    "images",
		Page:     "pages",
		Model:    "models",
		Cache:    "cache",
	},
	Root: wikifier.PageOptRoot{
		Wiki:     "", // aka /
		Image:    "/images",
		Category: "/topic",
		Page:     "/page",
		File:     "", // (i.e., disabled)
	},
	Image: wikifier.PageOptImage{
		Retina:     []int{2, 3},
		SizeMethod: "server",
		Rounding:   "normal",
		Calc:       defaultImageCalc,
		Sizer:      defaultImageSizer,
	},
	Category: wikifier.PageOptCategory{
		PerPage: 5,
	},
	Search: wikifier.PageOptSearch{
		Enable: true,
	},
	Link: wikifier.PageOptLink{
		ParseInternal: linkPageExists,
		ParseCategory: linkCategoryExists,
	},
	External: map[string]wikifier.PageOptExternal{
		"wp": wikifier.PageOptExternal{"Wikipedia", "https://en.wikipedia.org/wiki", wikifier.PageOptExternalTypeMediaWiki},
	},
}

func (w *Wiki) readConfig(file string) error {
	// create a Page for the configuration file
	// only compute the variables
	confPage := wikifier.NewPage(file)
	confPage.VarsOnly = true

	// set these variables for use in the config
	// FIXME: where does dir.wikifier come from?
	confPage.Set("dir.wiki", w.Opt.Dir.Wiki)
	confPage.Set("dir.wikifier", w.Opt.Dir.Wikifier)

	// parse the config
	if err := confPage.Parse(); err != nil {
		return errors.Wrap(err, "failed to parse configuration "+file)
	}

	// convert the config to wikifier.PageOpt
	if err := wikifier.InjectPageOpt(confPage, &w.Opt); err != nil {
		return err
	}

	return nil
}

func defaultImageCalc(name string, width, height int, page *wikifier.Page) (int, int) {
	path := page.Opt.Dir.Image + "/" + name
	bigW, bigH := getImageDimensions(path)

	// original has no dimensions??
	if bigW == 0 || bigH == 0 {
		return 0, 0
	}

	// requesting 0x0 is same as requesting full-size
	if width == 0 && height == 0 {
		return bigW, bigH
	}

	// determine missing dimension
	return calculateImageDimensions(bigW, bigH, width, height)
}

func defaultImageSizer(name string, width, height int, page *wikifier.Page) string {
	si := SizedImageFromName(name)
	si.Width = width
	si.Height = height
	return page.Opt.Root.Image + "/" + si.FullName()
}

func linkPageExists(page *wikifier.Page, ok *bool, target, tooltip, displayDefault *string) {
	w, good := page.Wiki.(*Wiki)
	if !good {
		return
	}

	// TODO: tons of crazy stuff for page prefixes
	// TODO: targets relative to page root
	pageName := wikifier.PageNameLink(*displayDefault, false)
	fmt.Println("T:", pageName)
	page.PageLinks[pageName] = append(page.PageLinks[pageName], 1) // FIXME: line number
	*ok = w.NewPage(pageName).Exists()
}

func linkCategoryExists(page *wikifier.Page, ok *bool, target, tooltip, displayDefault *string) {
	w, good := page.Wiki.(*Wiki)
	if !good {
		return
	}
	catName := wikifier.CategoryName(*displayDefault, false)
	*ok = w.GetCategory(catName).Exists()
}

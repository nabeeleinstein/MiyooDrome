package main

import (
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func (app *MiyooPod) isSearchableMenu() bool {
	if len(app.MenuStack) < 2 {
		return false
	}
	page := app.MenuStack[len(app.MenuStack)-1]
	if page.SearchScope != "" {
		return true
	}
	for _, item := range page.Items {
		if item.browseName() != "" {
			return true
		}
	}
	return false
}

func (item *MenuItem) browseName() string {
	switch {
	case item.Track != nil:
		return item.Track.Title
	case item.Album != nil:
		return item.Album.Name
	case item.Artist != nil:
		return item.Artist.Name
	case item.RemoteCover != nil:
		return item.RemoteCover.title
	default:
		return item.IndexName
	}
}
func sortBrowse(items []*MenuItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(items[i].browseName()) < strings.ToLower(items[j].browseName())
	})
}
func initial(item *MenuItem) rune {
	name := strings.TrimSpace(item.browseName())
	if name == "" {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(name)
	if !unicode.IsLetter(r) {
		return '#'
	}
	return unicode.ToUpper(r)
}
func (page *MenuScreen) letterTarget(step int) int {
	start := initial(page.Items[page.SelIndex])
	for i := page.SelIndex + step; i >= 0 && i < len(page.Items); i += step {
		letter := initial(page.Items[i])
		if letter == 0 || letter == start {
			continue
		}
		for i > 0 && initial(page.Items[i-1]) == letter {
			i--
		}
		return i
	}
	return -1
}

func (app *MiyooPod) repeatByLetter(now time.Time) bool {
	if (app.LastKey != DOWN && app.LastKey != UP) || app.CurrentScreen != ScreenMenu || len(app.MenuStack) == 0 {
		return false
	}
	page := app.MenuStack[len(app.MenuStack)-1]
	if !page.Alphabetical || len(page.Items) <= VISIBLE_ITEMS {
		return false
	}
	if now.Sub(app.LastKeyTime) < 700*time.Millisecond || now.Sub(app.LastRepeatTime) < 280*time.Millisecond {
		return true
	}
	app.LastRepeatTime = now
	step := 1
	if app.LastKey == UP {
		step = -1
	}
	target := page.letterTarget(step)
	if target < 0 && step == 1 && page.nextAlbums != nil {
		previousSize := len(page.Items)
		page.nextAlbums()
		target = page.letterTarget(step)
		if target < 0 && len(page.Items) > previousSize {
			target = previousSize - 1
		}
	}
	if target >= 0 {
		page.SelIndex = target
		page.adjustScroll()
		app.notify(string(initial(page.Items[target])))
	}
	app.drawCurrentScreen()
	return true
}

func (app *MiyooPod) notify(s string) {
	app.notice = s
	app.noticeUntil = time.Now().Add(1600 * time.Millisecond)
	app.MenuBG = nil
	app.requestRedraw()
}
func (app *MiyooPod) drawNotice() {
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.SelBG)
	dc.DrawRoundedRectangle(36, SCREEN_HEIGHT-103, 568, 49, 8)
	dc.Fill()
	dc.SetHexColor(app.CurrentTheme.SelTxt)
	dc.SetFontFace(app.FontMenu)
	dc.DrawStringAnchored(app.truncateText(app.notice, 540, app.FontMenu), 320, SCREEN_HEIGHT-78, .5, .5)
}

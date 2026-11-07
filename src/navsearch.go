package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/nabeeleinstein/MiyooDrome/internal/onionkeyboard"
	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

func (app *MiyooPod) openNavSearch() {
	app.beginSearch("all", A)
}
func (app *MiyooPod) searchButton(label, scope string) *MenuItem {
	return &MenuItem{Label: label, Action: func() { app.beginSearch(scope, A) }}
}
func (app *MiyooPod) beginSearch(scope string, release Key) {
	app.searchScope = scope
	app.navQuery = ""
	app.searchRelease = NONE
	if tryOnionKeyboard && onionkeyboard.Available(onionSystemDir) {
		app.searchRelease = release
	}
	app.navKeyboard = textKeyboard{qwerty: true}
	app.LastKey = NONE
	app.setScreen(ScreenNavSearch)
}
func (app *MiyooPod) drawNavSearch() {
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.BG)
	dc.Clear()
	app.drawHeader(app.searchTitle())
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.SetFontFace(app.FontHeader)
	dc.DrawStringAnchored("Enter a name or title", 24, 70, 0, 0.5)
	if app.searchRelease != NONE {
		dc.SetFontFace(app.FontMenu)
		dc.DrawStringAnchored("Opening Onion keyboard…", 320, 220, .5, .5)
		return
	}
	app.drawTextKeyboard(&app.navKeyboard, app.navQuery, "", "Search")
}
func (app *MiyooPod) handleNavSearchKey(key Key) {
	k := &app.navKeyboard
	if k.move(key) {
		return
	}
	switch key {
	case B, MENU:
		app.setScreen(ScreenMenu)
		app.LastKey = NONE
	case X:
		_, n := utf8.DecodeLastRuneInString(app.navQuery)
		app.navQuery = app.navQuery[:len(app.navQuery)-n]
	case A:
		app.navQuery += string(k.rows()[k.row][k.col])
	case START:
		q := strings.TrimSpace(app.navQuery)
		if q == "" {
			return
		}
		parent := app.MenuStack[len(app.MenuStack)-1]
		var results *MenuScreen
		if app.searchScope == "list" {
			results = &MenuScreen{Title: "Search: " + q, Parent: parent, Built: true}
			for _, item := range parent.Items {
				if item.browseName() != "" && strings.Contains(strings.ToLower(item.Label), strings.ToLower(q)) {
					results.Items = append(results.Items, item)
				}
			}
			if len(results.Items) == 0 {
				results.Items = []*MenuItem{{Label: "No matches"}}
			}
		} else {
			results = app.navSearchPage(parent, q, 0, app.searchScope)
			results.Items = results.Builder()
			results.Built = true
		}
		app.MenuStack = append(app.MenuStack, results)
		app.MenuBG = nil
		app.setScreen(ScreenMenu)
		app.LastKey = NONE
	}
}

func (app *MiyooPod) navSearchPage(parent *MenuScreen, q string, offset int, scope string) *MenuScreen {
	sub := &MenuScreen{Title: "Search: " + q, Parent: parent}
	sub.Builder = func() []*MenuItem {
		c, err := navClient()
		if err != nil {
			return app.navError(err.Error())
		}
		ctx, cancel := navCtx()
		defer cancel()
		hits, err := c.Search(ctx, q, offset)
		if err != nil {
			return app.navError(err.Error())
		}
		if scope == "artists" {
			hits.Songs = nil
			hits.Albums = nil
		} else if scope == "albums" {
			hits.Songs = nil
			hits.Artists = nil
		}
		var items []*MenuItem
		for _, a := range hits.Artists {
			item := app.navArtistItem(sub, a)
			item.Label = "Artist: " + item.Label
			items = append(items, item)
		}
		for _, a := range hits.Albums {
			item := app.navAlbumItem(sub, c, a)
			item.Label = "Album: " + item.Label
			items = append(items, item)
		}
		tracks := navTracks(c.MusicDir(MUSIC_ROOT), hits.Songs, false)
		for i, item := range app.buildTrackMenuItems(tracks) {
			item.Label = fmt.Sprintf("Song: %s - %s", tracks[i].Title, tracks[i].Artist)
			items = append(items, item)
		}
		if len(hits.Songs) == navcore.SearchPageSize || len(hits.Artists) == navcore.SearchPageSize || len(hits.Albums) == navcore.SearchPageSize {
			items = append(items, &MenuItem{Label: "More results...", HasSubmenu: true, Submenu: app.navSearchPage(sub, q, offset+navcore.SearchPageSize, scope)})
		}
		if len(items) == 0 {
			items = append(items, &MenuItem{Label: "No matches"})
		}
		return items
	}
	return sub
}

func (app *MiyooPod) searchTitle() string {
	switch app.searchScope {
	case "albums":
		return "Search albums"
	case "artists":
		return "Search artists"
	case "list":
		return "Search this list"
	default:
		return "Search Navidrome"
	}
}

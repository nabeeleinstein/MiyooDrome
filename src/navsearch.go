package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

func (app *MiyooPod) openNavSearch() {
	app.navKeyboard = textKeyboard{}
	app.LastKey = NONE
	app.setScreen(ScreenNavSearch)
}
func (app *MiyooPod) drawNavSearch() {
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.BG)
	dc.Clear()
	app.drawHeader("Search Navidrome")
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.SetFontFace(app.FontHeader)
	dc.DrawStringAnchored("Artist, album or song", 24, 70, 0, 0.5)
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
		app.navQuery += string(loginKeys[k.page][k.row][k.col])
	case START:
		q := strings.TrimSpace(app.navQuery)
		if q == "" {
			return
		}
		parent := app.MenuStack[len(app.MenuStack)-1]
		results := app.navSearchPage(parent, q, 0)
		results.Items = results.Builder()
		results.Built = true
		app.MenuStack = append(app.MenuStack, results)
		app.MenuBG = nil
		app.setScreen(ScreenMenu)
		app.LastKey = NONE
	}
}

func (app *MiyooPod) navSearchPage(parent *MenuScreen, q string, offset int) *MenuScreen {
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
			items = append(items, &MenuItem{Label: "More results...", HasSubmenu: true, Submenu: app.navSearchPage(sub, q, offset+navcore.SearchPageSize)})
		}
		if len(items) == 0 {
			items = append(items, &MenuItem{Label: "No matches"})
		}
		return items
	}
	return sub
}

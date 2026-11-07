package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

const navConfigPath = appDataDir + "/config.json"

func navClient() (*navcore.Client, error) {
	c, err := navcore.ReadConfig(navConfigPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Open Navidrome > Sign in to connect")
	}
	if err != nil {
		return nil, err
	}
	return navcore.New(c)
}

func (app *MiyooPod) navError(message string) []*MenuItem {
	logMsg("Navidrome: " + message)
	return []*MenuItem{{Label: "Navidrome unavailable", Action: func() { app.showError(message) }}}
}
func (app *MiyooPod) navRootItem(root *MenuScreen) *MenuItem {
	menu := &MenuScreen{ReloadOnOpen: true, Title: "Navidrome", Parent: root}
	menu.Builder = func() []*MenuItem {
		albums := &MenuScreen{ReloadOnOpen: true, Title: "Online Albums", Alphabetical: true, SearchScope: "albums", Parent: menu}
		albums.Builder = func() []*MenuItem { return app.navAlbumItems(albums, 0) }
		artists := &MenuScreen{ReloadOnOpen: true, Title: "Online Artists", Alphabetical: true, SearchScope: "artists", Parent: menu}
		artists.Builder = func() []*MenuItem { return app.navArtistItems(artists) }
		playlists := &MenuScreen{ReloadOnOpen: true, Title: "Online Playlists", Parent: menu}
		playlists.Builder = func() []*MenuItem { return app.navPlaylistItems(playlists) }
		connection := &MenuScreen{Title: "Test connection", Parent: menu, ReloadOnOpen: true, Builder: app.navConnection}
		items := []*MenuItem{
			{Label: "Albums", HasSubmenu: true, Submenu: albums},
			{Label: "Artists", HasSubmenu: true, Submenu: artists},
			{Label: "Playlists", HasSubmenu: true, Submenu: playlists},
			{Label: "Test connection", HasSubmenu: true, Submenu: connection},
		}
		signIn := &MenuItem{Label: "Sign in", Action: app.openSignIn}
		if _, err := navcore.ReadConfig(navConfigPath); err != nil {
			return append([]*MenuItem{signIn}, items...)
		}
		return append(items, signIn)
	}
	return &MenuItem{Label: "Navidrome", HasSubmenu: true, Submenu: menu}
}
func (app *MiyooPod) navConnection() []*MenuItem {
	client, err := navClient()
	if err == nil {
		ctx, cancel := navCtx()
		defer cancel()
		err = client.Ping(ctx)
	}
	if err != nil {
		return app.navError(err.Error())
	}
	return []*MenuItem{{Label: "Connected to Navidrome"}}
}

func navCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 14*time.Second)
}
func (app *MiyooPod) navAlbumItems(parent *MenuScreen, offset int) []*MenuItem {
	parent.nextAlbums = nil
	c, e := navClient()
	if e != nil {
		return app.navError(e.Error())
	}
	ctx, cancel := navCtx()
	defer cancel()
	albums, e := c.Albums(ctx, offset)
	if e != nil {
		return app.navError(e.Error())
	}
	items := make([]*MenuItem, 0, len(albums)+1)
	for _, album := range albums {
		items = append(items, app.navAlbumItem(parent, c, album))
	}
	if len(albums) == 100 {
		parent.nextAlbums = func() {
			retry := parent.nextAlbums
			more := app.navAlbumItems(parent, offset+len(albums))
			if len(more) == 1 && more[0].Label == "Navidrome unavailable" {
				parent.nextAlbums = retry
				more[0].Action()
				app.LastKey = NONE
				return
			}

			parent.Items = append(parent.Items[:len(parent.Items)-1], more...)
			parent.SelIndex = min(parent.SelIndex, len(parent.Items)-1)
			parent.adjustScroll()
			app.MenuBG = nil
		}
		items = append(items, &MenuItem{Label: "More albums...", Action: parent.nextAlbums})
	}
	if offset == 0 {
		items = append([]*MenuItem{app.searchButton("Search albums", "albums")}, items...)
	}
	return items
}
func (app *MiyooPod) navArtistItems(parent *MenuScreen) []*MenuItem {
	c, e := navClient()
	if e != nil {
		return app.navError(e.Error())
	}
	ctx, cancel := navCtx()
	defer cancel()
	artists, e := c.Artists(ctx)
	if e != nil {
		return app.navError(e.Error())
	}
	items := make([]*MenuItem, 0, len(artists))
	for _, artist := range artists {
		items = append(items, app.navArtistItem(parent, artist))
	}
	sortBrowse(items)
	return append([]*MenuItem{app.searchButton("Search artists", "artists")}, items...)
}
func (app *MiyooPod) navArtistAlbums(parent *MenuScreen, artist navcore.Artist) []*MenuItem {
	c, e := navClient()
	if e != nil {
		return app.navError(e.Error())
	}
	ctx, cancel := navCtx()
	defer cancel()
	albums, e := c.ArtistAlbums(ctx, artist.ID)
	if e != nil {
		return app.navError(e.Error())
	}
	if len(albums) == 0 {
		return []*MenuItem{{Label: "No albums found"}}
	}
	items := []*MenuItem{
		{Label: "Play all songs", Action: func() { app.navPlayArtist(c, artist, albums, false) }},
		{Label: "Shuffle artist", Action: func() { app.navPlayArtist(c, artist, albums, true) }},
		app.searchButton("Search these albums", "list"),
	}
	for _, album := range albums {
		items = append(items, app.navAlbumItem(parent, c, album))
	}
	return items
}
func (app *MiyooPod) navPlaylistItems(parent *MenuScreen) []*MenuItem {
	c, e := navClient()
	if e != nil {
		return app.navError(e.Error())
	}
	ctx, cancel := navCtx()
	defer cancel()
	playlists, e := c.Playlists(ctx)
	if e != nil {
		return app.navError(e.Error())
	}
	items := make([]*MenuItem, 0, len(playlists))
	for _, playlist := range playlists {
		p := playlist
		submenu := &MenuScreen{ReloadOnOpen: true, Title: p.Name, Parent: parent}
		submenu.Builder = func() []*MenuItem { return app.navPlaylistSongs(p.ID) }
		items = append(items, &MenuItem{Label: p.Name, HasSubmenu: true, Submenu: submenu})
	}
	if len(items) == 0 {
		return []*MenuItem{{Label: "No playlists found"}}
	}
	return items
}
func (app *MiyooPod) navPlaylistSongs(id string) []*MenuItem {
	c, err := navClient()
	if err != nil {
		return app.navError(err.Error())
	}
	ctx, cancel := navCtx()
	defer cancel()
	songs, err := c.PlaylistSongs(ctx, id)
	if err != nil {
		return app.navError(err.Error())
	}
	if len(songs) == 0 {
		return []*MenuItem{{Label: "Empty playlist"}}
	}
	return app.buildTrackMenuItems(navTracks(c.MusicDir(MUSIC_ROOT), songs, false))
}

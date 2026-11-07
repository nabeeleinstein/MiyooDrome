package main

// MiyooPod native menu integration for Navidrome.
// Navigation stays in upstream MiyooPod: A selects, B goes back.

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

func navClient() (*navcore.Client, error) {
	c, err := navcore.ReadConfig(appDataDir + "/config.json")
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Run configure.py on your computer to set up MiyooDrome")
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
		albums := &MenuScreen{ReloadOnOpen: true, Title: "Online Albums", Parent: menu}
		albums.Builder = func() []*MenuItem { return app.navAlbumItems(albums, 0) }
		artists := &MenuScreen{ReloadOnOpen: true, Title: "Online Artists", Parent: menu}
		artists.Builder = func() []*MenuItem { return app.navArtistItems(artists) }
		playlists := &MenuScreen{ReloadOnOpen: true, Title: "Online Playlists", Parent: menu}
		playlists.Builder = func() []*MenuItem { return app.navPlaylistItems(playlists) }
		connection := &MenuScreen{Title: "Test connection", Parent: menu, ReloadOnOpen: true, Builder: app.navConnection}
		return []*MenuItem{
			{Label: "Albums", HasSubmenu: true, Submenu: albums},
			{Label: "Artists", HasSubmenu: true, Submenu: artists},
			{Label: "Playlists", HasSubmenu: true, Submenu: playlists},
			{Label: "Test connection", HasSubmenu: true, Submenu: connection},
			{Label: "Rescan offline songs", Action: func() { app.rescanLibrary() }},
		}
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
		a := album
		songsMenu := &MenuScreen{ReloadOnOpen: true, Title: a.Name, Parent: parent}
		songsMenu.Builder = func() []*MenuItem { return app.navSongs(songsMenu, a.ID, false) }
		items = append(items, &MenuItem{Label: a.Name + " - " + a.Artist, HasSubmenu: true, Submenu: songsMenu})
	}
	if len(albums) == 100 {
		more := &MenuScreen{ReloadOnOpen: true, Title: fmt.Sprintf("Albums %d+", offset+100), Parent: parent}
		more.Builder = func() []*MenuItem { return app.navAlbumItems(more, offset+100) }
		items = append(items, &MenuItem{Label: "More albums...", HasSubmenu: true, Submenu: more})
	}
	if len(items) == 0 {
		return []*MenuItem{{Label: "No albums found"}}
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
		a := artist
		menu := &MenuScreen{ReloadOnOpen: true, Title: a.Name, Parent: parent}
		menu.Builder = func() []*MenuItem { return app.navArtistAlbums(menu, a.ID) }
		items = append(items, &MenuItem{Label: a.Name, HasSubmenu: true, Submenu: menu})
	}
	if len(items) == 0 {
		return []*MenuItem{{Label: "No artists found"}}
	}
	return items
}
func (app *MiyooPod) navArtistAlbums(parent *MenuScreen, id string) []*MenuItem {
	c, e := navClient()
	if e != nil {
		return app.navError(e.Error())
	}
	ctx, cancel := navCtx()
	defer cancel()
	albums, e := c.ArtistAlbums(ctx, id)
	if e != nil {
		return app.navError(e.Error())
	}
	items := make([]*MenuItem, 0, len(albums))
	for _, album := range albums {
		a := album
		submenu := &MenuScreen{ReloadOnOpen: true, Title: a.Name, Parent: parent}
		submenu.Builder = func() []*MenuItem { return app.navSongs(submenu, a.ID, false) }
		items = append(items, &MenuItem{Label: a.Name, HasSubmenu: true, Submenu: submenu})
	}
	if len(items) == 0 {
		return []*MenuItem{{Label: "No albums found"}}
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
		submenu.Builder = func() []*MenuItem { return app.navSongs(submenu, p.ID, true) }
		items = append(items, &MenuItem{Label: p.Name, HasSubmenu: true, Submenu: submenu})
	}
	if len(items) == 0 {
		return []*MenuItem{{Label: "No playlists found"}}
	}
	return items
}
func (app *MiyooPod) navSongs(parent *MenuScreen, id string, playlist bool) []*MenuItem {
	c, e := navClient()
	if e != nil {
		return app.navError(e.Error())
	}
	ctx, cancel := navCtx()
	defer cancel()
	var songs []navcore.Song
	if playlist {
		songs, e = c.PlaylistSongs(ctx, id)
	} else {
		songs, e = c.Songs(ctx, id)
	}
	if e != nil {
		return app.navError(e.Error())
	}
	items := make([]*MenuItem, 0, len(songs))
	for _, song := range songs {
		s := song
		items = append(items, &MenuItem{Label: s.Title, Action: func() { app.navPlay(s) }})
	}
	if len(items) == 0 {
		return []*MenuItem{{Label: "No songs found"}}
	}
	return items
}
func (app *MiyooPod) navPlay(s navcore.Song) {
	c, e := navClient()
	if e != nil {
		app.showError(e.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// Download/cache the full MP3 before playback: MiyooPod's SDL2_mixer
	// accepts local files, not live HTTP streams. This avoids the FFplay bug.
	dest, e := c.CacheSong(ctx, s, MUSIC_ROOT)
	if e != nil {
		app.showError("Download failed: " + e.Error())
		return
	}
	fileInfo, e := os.Stat(dest)
	if e != nil || fileInfo.IsDir() {
		app.showError("Cached file missing")
		return
	}
	track := &Track{Path: dest, Title: s.Title, Artist: s.Artist, Album: s.Album, AlbumArtist: s.Artist, TrackNum: s.Track, Duration: float64(s.Duration)}
	app.playTrackFromList([]*Track{track}, 0)
}

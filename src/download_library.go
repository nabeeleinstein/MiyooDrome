package main

import (
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

type offlineAlbum struct {
	Album
	saved   *navcore.SavedAlbum
	preview *navPreview
}

func submenu(title string, parent *MenuScreen, alpha bool, build func(*MenuScreen) []*MenuItem) *MenuItem {
	page := &MenuScreen{Title: title, Parent: parent, Alphabetical: alpha}
	page.Builder = func() []*MenuItem { return build(page) }
	return &MenuItem{Label: title, Submenu: page, HasSubmenu: true}
}

// Download records are authoritative for saved music. The scan index adds
// manually imported files and playlists without re-reading tags on every visit.
func (app *MiyooPod) downloadLibrary(home *MenuScreen) []*MenuItem {
	saved, err := navcore.DownloadedSongs(MUSIC_ROOT, SHARED_MUSIC_DIR)
	if err != nil {
		return []*MenuItem{{Label: "Cannot read downloaded songs", Action: func() { app.showError(err.Error()) }}}
	}
	savedAlbums, err := navcore.DownloadedAlbums(MUSIC_ROOT, SHARED_MUSIC_DIR)
	if err != nil {
		return []*MenuItem{{Label: "Cannot read downloaded albums", Action: func() { app.showError(err.Error()) }}}
	}
	var tracks []*Track
	albums := map[string]*offlineAlbum{}
	artists := map[string][]*Track{}
	paths := map[string]bool{}
	add := func(t *Track, key string, preview *navPreview) {
		tracks = append(tracks, t)
		artists[t.Artist] = append(artists[t.Artist], t)
		a := albums[key]
		if a == nil {
			a = &offlineAlbum{Album: Album{Name: t.Album, Artist: t.Artist}, preview: preview}
			if t.NavSong == nil && app.Library != nil {
				artist := t.albumArtistName()
				if local := app.Library.AlbumsByKey[artist+"|"+t.Album]; local != nil {
					a.Artist = local.Artist
					a.ArtPath = local.ArtPath
					a.ArtImg = local.ArtImg
				}
			}
			albums[key] = a
		}
		a.Tracks = append(a.Tracks, t)
	}
	for i := range savedAlbums {
		a := &savedAlbums[i]
		albums[a.Dir+"\x00"+a.Album.ID] = &offlineAlbum{Album: Album{Name: a.Album.Name, Artist: a.Album.Artist}, saved: a, preview: navAlbumPreview(a.Dir, a.Album)}
	}
	for _, s := range saved {
		t := navTracks(s.Dir, []navcore.Song{s.Song}, true)[0]
		paths[s.Path] = true
		id := s.Song.AlbumID
		if id == "" {
			id = t.Artist + "\x00" + t.Album
		}
		preview := trackPreview(t)
		add(t, s.Dir+"\x00"+id, &preview)
	}
	if app.Library != nil {
		for _, t := range app.Library.Tracks {
			if t.NavSong != nil || paths[t.Path] {
				continue
			}
			if _, err := os.Stat(t.Path); err != nil {
				continue
			}
			artist := t.albumArtistName()
			add(t, "local\x00"+artist+"\x00"+t.Album, nil)
		}
	}
	sort.SliceStable(tracks, func(i, j int) bool { return strings.ToLower(tracks[i].Title) < strings.ToLower(tracks[j].Title) })
	rows := []*MenuItem{
		submenu("Songs", home, true, func(p *MenuScreen) []*MenuItem { return app.offlineList("songs", app.buildTrackMenuItems(tracks)) }),
		submenu("Albums", home, true, func(p *MenuScreen) []*MenuItem {
			var items []*MenuItem
			for _, a := range albums {
				album := a
				item := submenu(album.Name+" - "+album.Artist, p, false, func(page *MenuScreen) []*MenuItem {
					if album.saved != nil {
						a := album.saved
						rows := app.navAlbumActions(nil, a.Dir, a.Album, a.Songs)
						return slices.Insert(rows, 1, app.deleteAlbumItem(home, page, a))
					}
					return append([]*MenuItem{app.searchButton("Search these songs", "list")}, app.localAlbumItems(&album.Album)...)
				})
				if album.saved != nil {
					item.DeleteAlbum = func(parent *MenuScreen) {
						confirm := app.deleteAlbumItem(home, parent, album.saved).Submenu
						confirm.Items = confirm.Builder()
						confirm.Built = true
						app.MenuStack = append(app.MenuStack, confirm)
						app.LastKey = NONE
					}
				}
				item.Album = &album.Album
				item.RemoteCover = album.preview
				item.Submenu.RemoteCover = album.preview
				items = append(items, item)
			}
			sortBrowse(items)
			return app.offlineList("albums", items)
		}),
		submenu("Artists", home, true, func(p *MenuScreen) []*MenuItem {
			var items []*MenuItem
			for name, group := range artists {
				name, group := name, group
				item := submenu(name, p, false, func(page *MenuScreen) []*MenuItem {
					rows := []*MenuItem{{Label: "Play all songs", Action: func() { app.playGroup(group, false) }}, {Label: "Shuffle artist", Action: func() { app.playGroup(group, true) }}, app.searchButton("Search these songs", "list")}
					return append(rows, app.buildTrackMenuItems(group)...)
				})
				item.IndexName = name
				item.ArtistTracks = group
				items = append(items, item)
			}
			sortBrowse(items)
			return app.offlineList("artists", items)
		}),
	}
	if app.Library != nil && len(app.Library.Playlists) > 0 {
		rows = append(rows, submenu("Playlists", home, false, func(p *MenuScreen) []*MenuItem { return app.buildPlaylistMenuItems(p) }))
	}
	if len(tracks) > 0 {
		rows = append(rows, &MenuItem{Label: "Shuffle all songs", Action: func() { app.playGroup(tracks, true) }})
	}
	return rows
}
func (app *MiyooPod) offlineList(kind string, items []*MenuItem) []*MenuItem {
	if len(items) == 0 {
		return []*MenuItem{{Label: "No " + kind + " on this device"}}
	}
	return append([]*MenuItem{app.searchButton("Search "+kind, "list")}, items...)
}

func (app *MiyooPod) deleteAlbumItem(home, album *MenuScreen, saved *navcore.SavedAlbum) *MenuItem {
	item := submenu("Delete downloaded album", album, false, func(confirm *MenuScreen) []*MenuItem {
		return []*MenuItem{
			{Label: "Cancel", Action: func() { app.MenuStack = app.MenuStack[:len(app.MenuStack)-1] }},
			{Label: "Delete album from shared library", Action: func() {
				if err := navcore.DeleteAlbum(MUSIC_ROOT, SHARED_MUSIC_DIR, saved.Dir, saved.Album.ID); err != nil {
					app.showError(err.Error())
					return
				}
				app.refreshAfterRemoval(home, saved.Dir)
				app.notify("Removed album download")
			}},
		}
	})
	item.Submenu.ReloadOnOpen = true
	item.Submenu.Title = "Remove " + saved.Album.Name + "?"
	return item
}

func (app *MiyooPod) refreshAfterRemoval(home *MenuScreen, dir string) {
	for i, page := range app.MenuStack {
		if page == home {
			app.MenuStack = app.MenuStack[:i+1]
			break
		}
	}
	// Queue entries can stream again if their local copy was removed.
	for _, t := range app.Queue.Tracks {
		if t.Offline && t.NavSong != nil && t.navDir() == dir {
			if _, err := os.Stat(navcore.PlayablePath(SHARED_MUSIC_DIR, dir, t.NavSong.ID)); os.IsNotExist(err) {
				t.Offline = false
			}
		}
	}
	app.savePlaybackState()
	home.Items = home.Builder()
	home.SelIndex = 0
	home.ScrollOff = 0
	app.MenuBG = nil
}

func (app *MiyooPod) confirmDeleteSong(parent *MenuScreen, track *Track) {
	for _, home := range app.MenuStack {
		if !home.DownloadRoot {
			continue
		}
		confirm := &MenuScreen{Title: "Remove " + track.Title + "?", Parent: parent, Built: true}
		confirm.Items = []*MenuItem{
			{Label: "Cancel", Action: func() { app.MenuStack = app.MenuStack[:len(app.MenuStack)-1] }},
			{Label: "Delete song from shared library", Action: func() {
				dir := track.navDir()
				if err := navcore.DeleteSong(MUSIC_ROOT, SHARED_MUSIC_DIR, dir, track.NavSong.ID); err != nil {
					app.showError(err.Error())
					return
				}
				app.refreshAfterRemoval(home, dir)
				app.notify("Removed song download")
			}},
		}
		app.LastKey = NONE
		app.MenuStack = append(app.MenuStack, confirm)
		return
	}
}

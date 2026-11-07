package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

func navTracks(dir string, songs []navcore.Song, offline bool) []*Track {
	tracks := make([]*Track, 0, len(songs))
	for i := range songs {
		s := &songs[i]
		tracks = append(tracks, &Track{NavDir: dir, Path: navcore.PlayablePath(SHARED_MUSIC_DIR, dir, s.ID), Title: s.Title, Artist: s.Artist, Album: s.Album, AlbumArtist: s.Artist, Duration: float64(s.Duration), TrackNum: s.Track, DiscNum: s.Disc, NavSong: s, Offline: offline})
	}
	return tracks
}

func (t *Track) navDir() string {
	if t.NavDir != "" {
		return t.NavDir
	}
	return filepath.Dir(t.Path)
}

func (t *Track) resolveNavPath() {
	if t.NavSong != nil {
		t.NavDir = t.navDir()
		t.Path = navcore.PlayablePath(SHARED_MUSIC_DIR, t.NavDir, t.NavSong.ID)
	}
}

func (app *MiyooPod) prepareNavTrack(t *Track) error {
	if t.NavSong == nil {
		return nil
	}
	t.resolveNavPath()
	if info, err := os.Stat(t.Path); err == nil && info.Size() >= 1024 && info.Size() <= navcore.MaxTrackBytes {
		now := time.Now()
		_ = os.Chtimes(t.Path, now, now)
		return nil
	}
	if t.Offline {
		return errors.New("Song missing on SD card. Download the album again.")
	}
	c, err := navClient()
	if err != nil {
		return err
	}
	dir := c.MusicDir(MUSIC_ROOT)
	if t.NavDir != dir {
		return errors.New("Sign in to the account that created this queue")
	}
	defer c.HTTP.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	t.Path, err = c.CacheForPlayback(ctx, MUSIC_ROOT, SHARED_MUSIC_DIR, *t.NavSong, app.cacheLimit(), app.activeMusicPath())
	return err
}

func (app *MiyooPod) playGroup(tracks []*Track, shuffle bool) {
	if len(tracks) == 0 {
		app.showError("No songs found")
		return
	}
	app.Queue.Shuffle = shuffle
	start := 0
	if shuffle {
		start = rand.Intn(len(tracks))
	}
	app.playTrackFromList(tracks, start)
}

func (app *MiyooPod) localAlbumItems(a *Album) []*MenuItem {
	items := app.buildTrackMenuItemsWithNumbers(a.Tracks, true)
	ordered := make([]*Track, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, item.Track)
	}
	return append([]*MenuItem{{Label: "Play album", Action: func() { app.playGroup(ordered, false) }}}, items...)
}

func (app *MiyooPod) navArtistItem(parent *MenuScreen, artist navcore.Artist) *MenuItem {
	sub := &MenuScreen{Title: artist.Name, Parent: parent, ReloadOnOpen: true}
	sub.Builder = func() []*MenuItem { return app.navArtistAlbums(sub, artist) }
	return &MenuItem{IndexName: artist.Name, Label: artist.Name, Submenu: sub, HasSubmenu: true}
}

func navAlbumPreview(dir string, album navcore.Album, offline bool) *navPreview {
	id := album.CoverArt
	if id == "" {
		id = album.ID
	}
	return &navPreview{dir: dir, id: id, title: album.Name, artist: album.Artist, offline: offline}
}

func (app *MiyooPod) navAlbumItem(parent *MenuScreen, c *navcore.Client, album navcore.Album) *MenuItem {
	dir := c.MusicDir(MUSIC_ROOT)
	preview := navAlbumPreview(dir, album, false)
	sub := &MenuScreen{Title: album.Name, Parent: parent, ReloadOnOpen: true, RemoteCover: preview}
	sub.Builder = func() []*MenuItem {
		ctx, cancel := navCtx()
		defer cancel()
		songs, err := c.AlbumSongs(ctx, album)
		if err != nil {
			return app.navError(err.Error())
		}
		if album.CoverArt == "" && len(songs) > 0 && songs[0].CoverArt != "" {
			preview.id = songs[0].CoverArt
		}
		return app.navAlbumActions(c, dir, album, songs)
	}
	return &MenuItem{Label: album.Name + " - " + album.Artist, Submenu: sub, HasSubmenu: true, RemoteCover: preview}
}

func (app *MiyooPod) navAlbumActions(c *navcore.Client, dir string, a navcore.Album, songs []navcore.Song) []*MenuItem {
	if len(songs) == 0 {
		return []*MenuItem{{Label: "Empty album"}}
	}
	// Stable order preserves the server's ordering when tags omit track numbers.
	sort.SliceStable(songs, func(i, j int) bool {
		if songs[i].Disc != songs[j].Disc {
			return songs[i].Disc < songs[j].Disc
		}
		return songs[i].Track < songs[j].Track
	})
	tracks := navTracks(dir, songs, c == nil)
	items := []*MenuItem{{Label: "Play album", Action: func() { app.playGroup(tracks, false) }}}
	if c != nil {
		items = append(items, &MenuItem{Label: "Download album", Action: func() { app.startAlbumDownload(c, a, songs) }})
	}
	items = append(items, app.searchButton("Search these songs", "list"))
	return append(items, app.buildTrackMenuItems(tracks)...)
}

func trackPreview(t *Track) navPreview {
	s := t.NavSong
	id := s.CoverArt
	if id == "" {
		id = s.AlbumID
	}
	if id == "" {
		id = s.ID
	}
	return navPreview{dir: t.navDir(), id: id, title: t.Album, artist: t.Artist, offline: t.Offline}
}

func (app *MiyooPod) navPlayArtist(c *navcore.Client, artist navcore.Artist, albums []navcore.Album, shuffle bool) {
	app.beginNavTask("Loading "+artist.Name, func(ctx context.Context, report func(string)) navTaskResult {
		songs, err := c.ArtistSongs(ctx, artist, albums)
		if err != nil {
			return navTaskResult{err: err}
		}
		if len(songs) == 0 {
			return navTaskResult{err: fmt.Errorf("No songs credited to %s", artist.Name)}
		}
		return navTaskResult{tracks: navTracks(c.MusicDir(MUSIC_ROOT), songs, false), shuffle: shuffle}
	})
}

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

func (app *MiyooPod) cacheLimit() int64 {
	if app.CacheLimitMiB == 128 || app.CacheLimitMiB == 512 {
		return int64(app.CacheLimitMiB) << 20
	}
	return navcore.DefaultCacheLimit
}
func (app *MiyooPod) activeMusicPath() string {
	if app.Playing != nil && app.Playing.Track != nil && app.Playing.State != StateStopped {
		return app.Playing.Track.Path
	}
	return ""
}
func (app *MiyooPod) selectedNavSong() *Track {
	if len(app.MenuStack) == 0 {
		return nil
	}
	m := app.MenuStack[len(app.MenuStack)-1]
	if m.SelIndex < 0 || m.SelIndex >= len(m.Items) {
		return nil
	}
	t := m.Items[m.SelIndex].Track
	if t != nil && t.NavSong != nil && !t.Offline {
		return t
	}
	return nil
}
func (app *MiyooPod) downloadSong(t *Track) {
	c, err := navClient()
	if err != nil {
		app.showError(err.Error())
		return
	}
	s := *t.NavSong
	if t.navDir() != c.MusicDir(MUSIC_ROOT) {
		app.showError("This song belongs to a different connection")
		return
	}
	app.beginNavTask("Download song", func(ctx context.Context, report func(string)) navTaskResult {
		report(s.Title)
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := c.SaveSong(ctx, MUSIC_ROOT, SHARED_MUSIC_DIR, s); err != nil {
			return navTaskResult{err: err}
		}
		id := trackPreview(t).id
		artCtx, stop := context.WithTimeout(ctx, 8*time.Second)
		defer stop()
		_, _ = navcore.Cover(artCtx, c, c.MusicDir(MUSIC_ROOT), id)
		return navTaskResult{message: "Saved to Media/Music/" + appFolder}
	})
}
func (app *MiyooPod) savedSongsMenu(root *MenuScreen) *MenuItem {
	page := &MenuScreen{Title: "Downloaded songs", Parent: root, ReloadOnOpen: true}
	page.Builder = func() []*MenuItem {
		saved, err := navcore.DownloadedSongs(MUSIC_ROOT, SHARED_MUSIC_DIR)
		if err != nil {
			return app.navError(err.Error())
		}
		var tracks []*Track
		for _, s := range saved {
			tracks = append(tracks, navTracks(s.Dir, []navcore.Song{s.Song}, true)...)
		}
		if len(tracks) == 0 {
			return []*MenuItem{{Label: "No downloaded songs"}}
		}
		return app.buildTrackMenuItems(tracks)
	}
	return &MenuItem{Label: page.Title, HasSubmenu: true, Submenu: page}
}
func (app *MiyooPod) storageMenu(root *MenuScreen) *MenuItem {
	page := &MenuScreen{Title: "Storage", Parent: root, ReloadOnOpen: true}
	page.Builder = func() []*MenuItem {
		used, err := navcore.CacheUsage(MUSIC_ROOT)
		if err != nil {
			return app.navError(err.Error())
		}
		return []*MenuItem{
			{Label: fmt.Sprintf("Playback cache: %.1f MiB", float64(used)/(1<<20))},
			{Label: fmt.Sprintf("Cache limit: %d MiB", app.cacheLimit()>>20), Action: func() {
				previous := app.CacheLimitMiB
				switch app.cacheLimit() >> 20 {
				case 128:
					app.CacheLimitMiB = 256
				case 256:
					app.CacheLimitMiB = 512
				default:
					app.CacheLimitMiB = 128
				}
				if err := app.saveSettings(); err != nil {
					app.CacheLimitMiB = previous
					app.showError("Could not save cache limit")
					return
				}
				if _, err := navcore.TrimCache(MUSIC_ROOT, app.cacheLimit(), app.activeMusicPath()); err != nil {
					app.showError("Could not trim cache")
				}
				page.Items = page.Builder()
				app.MenuBG = nil
			}},
			{Label: "Clear old tracks", Action: func() {
				removed, err := navcore.TrimCache(MUSIC_ROOT, 0, app.activeMusicPath())
				if err != nil {
					app.showError("Could not clear some tracks")
					return
				}
				page.Items = page.Builder()
				page.Items[0].Label = fmt.Sprintf("Cleared %.1f MiB • downloads kept", float64(removed)/(1<<20))
				app.MenuBG = nil
			}},
		}
	}
	return &MenuItem{Label: page.Title, HasSubmenu: true, Submenu: page}
}

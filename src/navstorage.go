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
func (app *MiyooPod) selectedMenuItem() *MenuItem {
	if len(app.MenuStack) == 0 {
		return nil
	}
	m := app.MenuStack[len(app.MenuStack)-1]
	if m.SelIndex < 0 || m.SelIndex >= len(m.Items) {
		return nil
	}
	return m.Items[m.SelIndex]
}

func (app *MiyooPod) selectedNavSong() *Track {
	item := app.selectedMenuItem()
	if item == nil {
		return nil
	}
	t := item.Track
	if t != nil && t.NavSong != nil {
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
		dir := c.MusicDir(MUSIC_ROOT)
		if _, err := navcore.ResolveCover(artCtx, c, dir, id, navcore.PlayablePath(SHARED_MUSIC_DIR, dir, s.ID), s.AlbumID, s.ID); err != nil {
			return navTaskResult{message: "Song saved. Artwork unavailable offline."}
		}
		return navTaskResult{message: "Saved to Media/Music/" + appFolder}
	})
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

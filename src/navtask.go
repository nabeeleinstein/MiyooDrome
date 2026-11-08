package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/fogleman/gg"
	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

type navTaskResult struct {
	message string
	err     error
	tracks  []*Track
	shuffle bool
}
type navTask struct {
	previous      ScreenType
	title, status string
	done          bool
	cancel        context.CancelFunc
	updates       chan string
	result        chan navTaskResult
}

func (app *MiyooPod) beginNavTask(title string, work func(context.Context, func(string)) navTaskResult) {
	if app.navTask != nil {
		app.setScreen(ScreenNavTask)
		app.requestRedraw()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	t := &navTask{previous: app.CurrentScreen, title: title, status: "Starting...", cancel: cancel, updates: make(chan string, 1), result: make(chan navTaskResult, 1)}
	app.navTask = t
	app.setScreen(ScreenNavTask)
	app.LastKey = NONE
	go func() {
		defer cancel()
		t.result <- work(ctx, func(status string) {
			select {
			case t.updates <- status:
			case <-ctx.Done():
			}
		})
	}()
}

func (app *MiyooPod) startAlbumDownload(c *navcore.Client, album navcore.Album, songs []navcore.Song) {
	app.beginNavTask("Download album", func(ctx context.Context, report func(string)) navTaskResult {
		artSaved, err := c.DownloadAlbumWithArtwork(ctx, MUSIC_ROOT, SHARED_MUSIC_DIR, album, songs, func(i int, title string) { report(fmt.Sprintf("%d / %d   %s", i+1, len(songs), title)) })
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		if err == nil && !artSaved {
			return navTaskResult{message: "Album saved. Artwork unavailable offline."}
		}
		return navTaskResult{err: err}
	})
}

func (app *MiyooPod) pollNavUpdates() {
	a := &app.navArtwork
	select {
	case v := <-a.pending:
		a.pending = nil
		fresh := []navArtImage{v}
		for _, old := range a.cache {
			if old.key != v.key && len(fresh) < 5 {
				fresh = append(fresh, old)
			}
		}
		a.cache = fresh
		app.NPCacheDirty = true
		app.MenuBG = nil
		app.requestRedraw()
	default:
	}
	t := app.navTask
	if t == nil || t.done {
		return
	}
	select {
	case t.status = <-t.updates:
		app.requestRedraw()
	default:
	}
	select {
	case r := <-t.result:
		t.done = true
		// Retry previously missing covers after a download has populated the card.
		if r.err == nil && len(r.tracks) == 0 {
			app.navArtwork = navArtworkState{}
			app.NPCacheDirty = true
			app.MenuBG = nil
		}
		switch {
		case errors.Is(r.err, context.Canceled):
			t.status = "Cancelled"
		case r.err != nil:
			t.status = r.err.Error()
		case len(r.tracks) > 0:
			app.navTask = nil
			app.playGroup(r.tracks, r.shuffle)
			return
		case r.message != "":
			t.status = r.message
		default:
			t.status = "Saved to Downloaded library"
		}
		app.requestRedraw()
	default:
	}
}

func (app *MiyooPod) handleNavTaskKey(key Key) {
	t := app.navTask
	if t == nil {
		return
	}
	if t.done && (key == A || key == B || key == MENU) {
		app.navTask = nil
		app.setScreen(t.previous)
		app.LastKey = NONE
		return
	}
	if !t.done && (key == B || key == MENU) {
		t.cancel()
		t.status = "Cancelling..."
	}
}
func (app *MiyooPod) drawNavTask() {
	t := app.navTask
	if t == nil {
		return
	}
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.BG)
	dc.Clear()
	app.drawHeader(t.title)
	dc.SetFontFace(app.FontMenu)
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.DrawStringWrapped(t.status, 320, 205, 0.5, 0.5, 590, 1.4, gg.AlignCenter)
	action := "Cancel"
	if t.done {
		action = "Back"
	}
	app.drawButtonLegend(20, 440, "B", action)
}

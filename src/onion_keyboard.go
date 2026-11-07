package main

import (
	"context"
	"github.com/nabeeleinstein/MiyooDrome/internal/onionkeyboard"
	"strings"
)

// Onion's SDL 1 keyboard looks up its fonts and key images relative to this directory.
const onionSystemDir = "/mnt/SDCARD/.tmp_update"

type onionKeyboardSession struct {
	cancel context.CancelFunc
	done   chan onionkeyboard.Result
}

func (app *MiyooPod) startOnionKeyboard() {
	f := &app.signIn
	if app.keyboardSession != nil {
		return
	}
	var title, initial string
	if app.CurrentScreen == ScreenNavSearch {
		title, initial = app.searchTitle(), app.navQuery
	} else {
		if !f.usesOnionKeyboard() {
			return
		}
		title, initial = "Server address", f.server
		if f.field == loginUser {
			title, initial = "Username", f.user
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &onionKeyboardSession{cancel: cancel, done: make(chan onionkeyboard.Result, 1)}
	app.keyboardSession = s
	app.keyboardActive.Store(true)
	app.LastKey = NONE
	app.displayMu.Lock()
	suspendPlayerVideo()
	go func() { s.done <- onionkeyboard.Edit(ctx, onionSystemDir, title, initial) }()
}

func (app *MiyooPod) finishOnionKeyboard(result onionkeyboard.Result) {
	app.keyboardSession.cancel()
	app.keyboardSession = nil
	restored := resumePlayerVideo()
	app.resetInactivityTimer()
	app.keyboardActive.Store(false)
	app.displayMu.Unlock()
	if !restored {
		logMsg("ERROR: Could not reopen the display after Onion keyboard")
		app.Running = false
		return
	}
	if app.CurrentScreen == ScreenNavSearch {
		switch {
		case result.Err != nil:
			app.notify("Using the built-in keyboard")
		case result.Accepted && strings.TrimSpace(result.Value) != "":
			app.navQuery = result.Value
			app.handleNavSearchKey(START)
		default:
			app.setScreen(ScreenMenu)
		}
		app.drawCurrentScreen()
		return
	}
	f := &app.signIn
	switch {
	case result.Err != nil:
		f.native = false
		f.message = "Onion keyboard unavailable. Use the keys below."
	case result.Accepted:
		if f.field == loginServer {
			f.server = result.Value
		} else {
			f.user = result.Value
		}
		app.handleSignInKey(START)
	}
	app.drawCurrentScreen()
}

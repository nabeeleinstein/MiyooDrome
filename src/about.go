package main

import (
	"time"

	"github.com/skip2/go-qrcode"
)

func (app *MiyooPod) showAboutScreen() {
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.BG)
	dc.Clear()
	dc.SetFontFace(app.FontTitle)
	dc.SetHexColor(app.CurrentTheme.HeaderTxt)
	dc.DrawStringAnchored(appName+" "+APP_VERSION+buildSuffix, SCREEN_WIDTH/2, 38, 0.5, 0.5)
	dc.SetFontFace(app.FontSmall)
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.DrawStringAnchored("Navidrome player maintained by nabeeleinstein", SCREEN_WIDTH/2, 85, 0.5, 0.5)
	dc.SetFontFace(app.FontMenu)
	dc.DrawStringAnchored("Based on MiyooPod by "+MIYOOPOD_AUTHOR, SCREEN_WIDTH/2, 146, 0.5, 0.5)
	dc.SetFontFace(app.FontSmall)
	dc.DrawStringAnchored("Original interface, controls and playback engine", SCREEN_WIDTH/2, 183, 0.5, 0.5)
	dc.DrawStringAnchored("Copyright 2026 Danilo Fragoso · MIT License", SCREEN_WIDTH/2, 214, 0.5, 0.5)
	if qr, err := qrcode.New(MIYOOPOD_URL, qrcode.Medium); err == nil {
		const size = 140
		app.fastBlitImage(qr.Image(size), (SCREEN_WIDTH-size)/2, 240)
	}
	dc.DrawStringAnchored("github.com/danfragoso/miyoopod", SCREEN_WIDTH/2, 402, 0.5, 0.5)
	dc.SetHexColor(app.CurrentTheme.Dim)
	dc.DrawStringAnchored("Press B to return", SCREEN_WIDTH/2, SCREEN_HEIGHT-20, 0.5, 0.5)
	app.triggerRefresh()
	app.waitForAboutExit()
}

// waitForAboutExit waits for the user to press B to exit the about screen
func (app *MiyooPod) waitForAboutExit() {
	for app.Running {
		key := Key(C_GetKeyPress())
		if key == NONE {
			time.Sleep(33 * time.Millisecond)
			continue
		}

		if key == B || key == MENU {
			// Return to menu. Clear any armed key-repeat: the key that opened
			// this screen had its release consumed by this loop, so the main
			// loop must not auto-repeat it (which would reopen About).
			app.LastKey = NONE
			app.setScreen(ScreenMenu)
			app.drawMenuScreen()
			app.triggerRefresh()
			return
		}
	}
}

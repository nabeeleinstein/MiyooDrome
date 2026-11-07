package main

import (
	"github.com/skip2/go-qrcode"
)

func (app *MiyooPod) drawAboutScreen() {
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.BG)
	dc.Clear()
	dc.SetFontFace(app.FontTitle)
	dc.SetHexColor(app.CurrentTheme.HeaderTxt)
	dc.DrawStringAnchored(appName+" "+APP_VERSION+buildSuffix, SCREEN_WIDTH/2, 38, 0.5, 0.5)
	dc.SetFontFace(app.FontSmall)
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.DrawStringAnchored("Your Navidrome music on Miyoo Mini", 320, 97, .5, .5)
	dc.DrawStringAnchored("Browse your server, build a queue and save music for offline play.", 320, 135, .5, .5)
	dc.DrawStringAnchored("Maintained by nabeeleinstein", 320, 174, .5, .5)
	if qr, err := qrcode.New(projectURL, qrcode.Medium); err == nil {
		const size = 98
		app.fastBlitImage(qr.Image(size), (SCREEN_WIDTH-size)/2, 200)
	}
	dc.DrawStringAnchored("github.com/nabeeleinstein/MiyooDrome", 320, 320, .5, .5)
	dc.DrawStringAnchored("Based on MiyooPod by "+MIYOOPOD_AUTHOR+" · MIT License", 320, 377, .5, .5)
	dc.SetHexColor(app.CurrentTheme.Dim)
	dc.DrawStringAnchored("B   Back       R2   Now Playing", 320, 447, .5, .5)
}

package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/fogleman/gg"
)

// drawLogo draws a front-facing Miyoo Mini with a record and paired grooves
// inspired by the Navidrome mark. Coordinates use a 64px launcher canvas.
func drawLogo(dc *gg.Context, x, y, scale float64, accent string) {
	dc.Push()
	defer dc.Pop()
	dc.Translate(x, y)
	dc.Scale(scale, scale)
	dc.Translate(-32, -32)

	const ink = "#222437"
	// A light rim keeps the silhouette visible against a dark Onion theme.
	dc.DrawRoundedRectangle(9, 1, 46, 62, 7)
	dc.SetHexColor("#f3efdf")
	dc.Fill()
	dc.DrawRoundedRectangle(10, 2, 44, 60, 6)
	dc.SetHexColor(ink)
	dc.Fill()
	dc.DrawRoundedRectangle(11.5, 3.5, 41, 56.5, 4.8)
	dc.SetHexColor("#e3dfcd")
	dc.Fill()

	dc.DrawRoundedRectangle(13, 6, 38, 30, 3)
	dc.SetHexColor(ink)
	dc.Fill()
	dc.DrawRoundedRectangle(15, 8, 34, 26, 1.8)
	display := gg.NewLinearGradient(15, 8, 49, 34)
	display.AddColorStop(0, color.NRGBA{R: 23, G: 31, B: 71, A: 255})
	display.AddColorStop(1, color.NRGBA{R: 56, G: 33, B: 89, A: 255})
	dc.SetFillStyle(display)
	dc.Fill()

	// Vinyl face, opposite groove arcs and spindle, simplified for a 24px display.
	dc.DrawCircle(32, 21, 11.3)
	dc.SetHexColor("#60e2e0")
	dc.Fill()
	dc.DrawCircle(32, 21, 9.7)
	dc.SetHexColor("#303055")
	dc.Fill()
	dc.SetHexColor("#99faf0")
	dc.SetLineWidth(1.45 * scale)
	dc.SetLineCapRound()
	dc.DrawArc(32, 21, 7.5, 3.31, 4.82)
	dc.Stroke()
	dc.SetHexColor("#ff9ad6")
	dc.DrawArc(32, 21, 7.5, 0.17, 1.68)
	dc.Stroke()
	dc.DrawCircle(32, 21, 4.25)
	dc.SetHexColor("#c1b1ff")
	dc.Fill()
	dc.DrawCircle(32, 21, 1.45)
	dc.SetHexColor(ink)
	dc.Fill()

	// D-pad, face buttons, and the two small function keys.
	dc.DrawRoundedRectangle(17, 42, 12, 4.8, 1)
	dc.DrawRoundedRectangle(20.6, 38.4, 4.8, 12, 1)
	dc.SetHexColor(ink)
	dc.Fill()
	for _, p := range [][2]float64{{40, 40}, {44.6, 44.4}, {35.4, 44.4}, {40, 48.8}} {
		dc.DrawCircle(p[0], p[1], 2.55)
		dc.SetHexColor("#80374c")
		dc.Fill()
	}
	dc.SetHexColor("#797579")
	dc.DrawRoundedRectangle(25, 54, 5.1, 1.9, .9)
	dc.DrawRoundedRectangle(33, 54, 5.1, 1.9, .9)
	dc.Fill()
	dc.DrawCircle(48, 4.8, .85)
	dc.SetHexColor(accent)
	dc.Fill()
}

// generateIconPNG writes the same mark used by the launcher and splash.
func (app *MiyooPod) generateIconPNG() error {
	const iconSize = 64
	iconDC := gg.NewContext(iconSize, iconSize)
	drawLogo(iconDC, 32, 32, 1, app.CurrentTheme.Accent)

	// Ensure assets directory exists
	assetsDir := "./assets"
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets directory: %v", err)
	}

	// Save to file
	iconPath := assetsDir + "/icn.png"
	file, err := os.Create(iconPath)
	if err != nil {
		return fmt.Errorf("failed to create icon file: %v", err)
	}
	defer file.Close()

	img := iconDC.Image()
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("failed to encode icon PNG: %v", err)
	}

	logMsg(fmt.Sprintf("Generated icon: %s", iconPath))
	return nil
}

// drawLogoSplash draws the logo on the splash screen with theme-tinted background
func (app *MiyooPod) drawLogoSplash() {
	// Clear with theme background
	app.DC.SetHexColor(app.CurrentTheme.BG)
	app.DC.Clear()

	// Draw the logo in the upper portion
	drawLogo(app.DC, SCREEN_WIDTH/2, SCREEN_HEIGHT/2-60, 3.2, app.CurrentTheme.Accent)

	// Draw "MiyooDrome" title below the logo
	app.DC.SetFontFace(app.FontTitle)
	app.DC.SetHexColor(app.CurrentTheme.HeaderTxt)
	app.DC.DrawStringWrapped(appName, SCREEN_WIDTH/2, SCREEN_HEIGHT/2+100, 0.5, 0.5, 600, 1.5, gg.AlignCenter)

	// Draw "Loading..." text
	app.DC.SetFontFace(app.FontSmall)
	app.DC.SetHexColor(app.CurrentTheme.Dim)
	app.DC.DrawStringWrapped("Loading...", SCREEN_WIDTH/2, SCREEN_HEIGHT/2+140, 0.5, 0.5, 600, 1.5, gg.AlignCenter)

	app.triggerRefresh()
}

package main

import "unicode/utf8"

type textKeyboard struct{ page, row, col int }

func (f *textKeyboard) move(key Key) bool {
	rows := loginKeys[f.page]
	switch key {
	case UP, DOWN:
		step := 1
		if key == UP {
			step = -1
		}
		f.row = (f.row + len(rows) + step) % len(rows)
		f.col = min(f.col, len(rows[f.row])-1)
	case LEFT, RIGHT:
		width := len(rows[f.row])
		step := 1
		if key == LEFT {
			step = -1
		}
		f.col = (f.col + width + step) % width
	case Y:
		f.page = (f.page + 1) % len(loginKeys)
		f.col = min(f.col, len(loginKeys[f.page][f.row])-1)
	default:
		return false
	}
	return true
}

func (app *MiyooPod) drawTextKeyboard(f *textKeyboard, value, hint, next string) {
	dc, theme := app.DC, app.CurrentTheme
	dc.SetHexColor(theme.ProgBG)
	dc.DrawRoundedRectangle(20, 91, 600, 44, 6)
	dc.Fill()
	dc.SetFontFace(app.FontMenu)
	dc.SetHexColor(theme.ItemTxt)
	for len(value) > 0 {
		if w, _ := dc.MeasureString(value + "_"); w <= 573 {
			break
		}
		_, n := utf8.DecodeRuneInString(value)
		value = value[n:]
	}
	dc.DrawStringAnchored(value+"_", 32, 112, 0, 0.5)

	dc.SetFontFace(app.FontSmall)
	dc.DrawStringAnchored(app.truncateText(hint, 595, app.FontSmall), 24, 154, 0, 0.5)
	for r, row := range loginKeys[f.page] {
		for c, ch := range row {
			x, y := float64(20+60*c), float64(179+43*r)
			dc.SetFontFace(app.FontMenu)
			dc.SetHexColor(theme.ItemTxt)
			if r == f.row && c == f.col {
				dc.SetHexColor(theme.SelBG)
				dc.DrawRoundedRectangle(x, y, 56, 39, 5)
				dc.Fill()
				dc.SetHexColor(theme.SelTxt)
			}
			label := string(ch)
			if ch == ' ' {
				label = "SP"
				dc.SetFontFace(app.FontSmall)
			}
			dc.DrawStringAnchored(label, x+28, y+20, 0.5, 0.5)
		}
	}

	dc.SetFontFace(app.FontSmall)
	app.drawButtonLegend(24, 384, "A", "Type")
	app.drawButtonLegend(180, 384, "X", "Delete")
	app.drawButtonLegend(363, 384, "Y", "abc / ABC / #")
	app.drawButtonLegend(24, 438, "B", "Back")
	app.drawButtonLegend(180, 438, "MENU", "Cancel")
	app.drawButtonLegend(363, 438, "START", next)
}

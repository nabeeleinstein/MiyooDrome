package main

import (
	"strings"
	"unicode/utf8"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
)

const (
	loginServer = iota
	loginUser
	loginPassword
	loginDone
)

// The three pages cover all printable ASCII, including space.
var loginKeys = [][4]string{
	{"1234567890", "abcdefghij", "klmnopqrst", "uvwxyz./: "},
	{"1234567890", "ABCDEFGHIJ", "KLMNOPQRST", "UVWXYZ./: "},
	{"!@#$%^&*()", "-_=+[]{}<>", ".,:;/?\\|\"'", "~` "},
}

type signInForm struct {
	server, user, message string
	password              []byte
	field, page, row, col int
	bitrate               int
}

func (app *MiyooPod) openSignIn() {
	f := signInForm{server: "https://", bitrate: 192}
	if cfg, err := navcore.ReadConfig(navConfigPath); err == nil {
		f.server, f.user, f.bitrate = cfg.Server, cfg.Username, cfg.MaxBitrate
	}
	app.signIn = f
	app.LastKey = NONE
	app.setScreen(ScreenSignIn)
}

func (app *MiyooPod) closeSignIn() {
	clear(app.signIn.password)
	app.signIn = signInForm{}
	app.LastKey = NONE
	app.setScreen(ScreenMenu)
	// Rebuild this menu so Albums is available first after a new login.
	menu := app.MenuStack[len(app.MenuStack)-1]
	menu.Items = menu.Builder()
	menu.SelIndex, menu.ScrollOff = 0, 0
	app.MenuBG = nil
}

func (f *signInForm) edit(ch byte, erase bool) {
	if f.field == loginPassword {
		if erase {
			if n := len(f.password); n > 0 {
				f.password[n-1] = 0
				f.password = f.password[:n-1]
			}
		} else {
			f.password = append(f.password, ch)
		}
		return
	}
	value := &f.server
	if f.field == loginUser {
		value = &f.user
	}
	if erase {
		_, n := utf8.DecodeLastRuneInString(*value)
		*value = (*value)[:len(*value)-n]
	} else {
		*value += string(ch)
	}
}

func (app *MiyooPod) handleSignInKey(key Key) {
	f := &app.signIn
	if f.field == loginDone {
		if key == A || key == START || key == MENU || key == B {
			app.closeSignIn()
		}
		return
	}
	rows := loginKeys[f.page]
	switch key {
	case MENU:
		app.closeSignIn()
	case B:
		if f.field == loginServer {
			app.closeSignIn()
			return
		}
		f.field--
		f.message = ""
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
	case A, X:
		f.edit(rows[f.row][f.col], key == X)
		f.message = ""
	case L:
		if f.field == loginServer {
			if strings.HasPrefix(f.server, "https://") {
				f.server = "http://" + strings.TrimPrefix(f.server, "https://")
			} else {
				f.server = "https://" + strings.TrimPrefix(f.server, "http://")
			}
			f.message = ""
		}
	case START:
		switch f.field {
		case loginServer:
			f.server = strings.TrimSpace(f.server)
			if navcore.ValidateServer(f.server) != nil {
				f.message = "Enter an http:// or https:// server address."
				return
			}
		case loginUser:
			if strings.TrimSpace(f.user) == "" {
				f.message = "Enter your Navidrome username."
				return
			}
		case loginPassword:
			if len(f.password) == 0 {
				f.message = "Enter your Navidrome password."
				return
			}
			f.message = "Checking connection..."
			app.drawCurrentScreen()
			ctx, cancel := navCtx()
			defer cancel()
			if err := navcore.SignIn(ctx, navConfigPath, f.server, f.user, f.password, f.bitrate); err != nil {
				f.message = err.Error()
				return
			}
			clear(f.password)
			f.password = nil
		}
		f.field++
		f.page, f.row, f.col = 0, 0, 0
		f.message = ""
	}
}

func (app *MiyooPod) drawSignIn() {
	f, dc, theme := &app.signIn, app.DC, app.CurrentTheme
	dc.SetHexColor(theme.BG)
	dc.Clear()
	app.drawHeader("Navidrome sign-in")
	if f.field == loginDone {
		dc.SetFontFace(app.FontTitle)
		dc.SetHexColor(theme.ItemTxt)
		dc.DrawStringAnchored("Signed in to Navidrome", 320, 185, 0.5, 0.5)
		dc.SetFontFace(app.FontMenu)
		dc.DrawStringAnchored("Press A to browse music", 320, 235, 0.5, 0.5)
		return
	}

	labels := [...]string{"Server address  (1/3)", "Username  (2/3)", "Password  (3/3)"}
	dc.SetHexColor(theme.ItemTxt)
	dc.SetFontFace(app.FontHeader)
	dc.DrawStringAnchored(labels[f.field], 24, 70, 0, 0.5)
	value := f.server
	if f.field == loginUser {
		value = f.user
	}
	if f.field == loginPassword {
		value = strings.Repeat("*", min(len(f.password), 42))
	}
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
	hint := f.message
	if hint == "" && f.field == loginServer {
		hint = "L: HTTP/HTTPS     e.g. http://192.168.1.20:4533"
	}
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

	next := "Next"
	if f.field == loginPassword {
		next = "Sign in"
	}
	dc.SetFontFace(app.FontSmall)
	app.drawButtonLegend(24, 384, "A", "Type")
	app.drawButtonLegend(180, 384, "X", "Delete")
	app.drawButtonLegend(363, 384, "Y", "abc / ABC / #")
	app.drawButtonLegend(24, 438, "B", "Back")
	app.drawButtonLegend(180, 438, "MENU", "Cancel")
	app.drawButtonLegend(363, 438, "START", next)
}

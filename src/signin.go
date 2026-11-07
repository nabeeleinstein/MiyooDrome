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
	field                 int
	textKeyboard
	bitrate int
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
	if f.move(key) {
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
	hint := f.message
	if hint == "" && f.field == loginServer {
		hint = "L: HTTP/HTTPS     e.g. http://192.168.1.20:4533"
	}
	next := "Next"
	if f.field == loginPassword {
		next = "Sign in"
	}
	app.drawTextKeyboard(&f.textKeyboard, value, hint, next)
}

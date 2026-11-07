package onionkeyboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyboardCommand(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if Available(dir) {
		t.Fatal("absent keyboard detected")
	}
	t.Setenv("SDL_VIDEODRIVER", "mmiyoo")
	t.Setenv("SDL_AUDIODRIVER", "mmiyoo")
	t.Setenv("EGL_VIDEODRIVER", "mmiyoo")
	t.Setenv("EXPECTED_DIR", dir)
	t.Setenv("EXPECTED_VALUE", "name with spaces & punctuation")
	script := `#!/bin/sh
[ "$(pwd -P)" = "$(cd "$EXPECTED_DIR" && pwd -P)" ] && [ -z "$SDL_VIDEODRIVER$SDL_AUDIODRIVER$EGL_VIDEODRIVER" ] || exit 99
[ "$1" = -t ] && [ "$2" = Username ] && [ "$3" = -i ] && [ "$4" = "$EXPECTED_VALUE" ] && [ "$#" = 4 ] || exit 98
case "$SCENARIO" in
  confirm) printf 'diagnostic\n\nRESULT:\n%s\n' "$4";;
  empty) printf '\n\nRESULT:\n\n';;
  cancel) exit 1;;
  broken) exit 127;;
  malformed) printf '\n\nRESULT:\nfirst\nsecond\n';;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "kbinput"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if !Available(dir) {
		t.Fatal("installed keyboard missing")
	}
	for _, tc := range []struct {
		scenario string
		want     Result
		fail     bool
	}{
		{"confirm", Result{Value: os.Getenv("EXPECTED_VALUE"), Accepted: true}, false},
		{"empty", Result{Accepted: true}, false}, {"cancel", Result{}, false}, {"broken", Result{}, true}, {"malformed", Result{}, true},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			t.Setenv("SCENARIO", tc.scenario)
			got := Edit(context.Background(), dir, "Username", os.Getenv("EXPECTED_VALUE"))
			if (got.Err != nil) != tc.fail || got.Value != tc.want.Value || got.Accepted != tc.want.Accepted {
				t.Fatal(got)
			}
		})
	}
}

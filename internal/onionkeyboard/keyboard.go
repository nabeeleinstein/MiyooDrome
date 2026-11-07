package onionkeyboard

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func Available(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "bin", "kbinput"))
	return err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0111 != 0
}

type Result struct {
	Value    string
	Accepted bool
	Err      error
}

// Only the URL and username enter this function. Password input stays in the player.
func Edit(ctx context.Context, dir, title, initial string) Result {
	cmd := exec.CommandContext(ctx, filepath.Join(dir, "bin", "kbinput"), "-t", title, "-i", initial)
	cmd.Dir = dir
	// The player selects SDL2's mmiyoo driver. The Onion helper uses the system SDL 1 driver.
	for _, pair := range os.Environ() {
		name, _, _ := strings.Cut(pair, "=")
		switch name {
		case "SDL_VIDEODRIVER", "SDL_AUDIODRIVER", "EGL_VIDEODRIVER", "LD_LIBRARY_PATH":
			continue
		}
		cmd.Env = append(cmd.Env, pair)
	}
	cmd.Env = append(cmd.Env, "LD_LIBRARY_PATH="+filepath.Join(dir, "lib")+":/mnt/SDCARD/miyoo/lib:/customer/lib")
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return Result{Err: ctx.Err()}
	}
	var exit *exec.ExitError
	if err != nil && (!errors.As(err, &exit) || exit.ExitCode() < 0) {
		return Result{Err: err}
	}
	// kbinput emits a RESULT line only on confirmation. Preserve spaces in usernames.
	_, value, found := bytes.Cut(out, []byte("\nRESULT:\n"))
	if found {
		value = bytes.TrimSuffix(value, []byte("\n"))
		if bytes.ContainsAny(value, "\r\n\x00") || !utf8.Valid(value) {
			return Result{Err: errors.New("invalid keyboard response")}
		}
		return Result{Value: string(value), Accepted: true}
	}
	if err == nil || exit.ExitCode() == 1 {
		return Result{}
	}
	return Result{Err: errors.New("keyboard returned no result")}
}

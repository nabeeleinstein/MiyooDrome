package navcore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Version 0.3.0 pins contain just the embedded Song fields.
type downloadPin struct {
	Song
	Filename string `json:"filename,omitempty"`
	Pending  bool   `json:"pending,omitempty"`
}

func readPin(private, id string) (downloadPin, error) {
	b, err := os.ReadFile(pinPath(private))
	if err != nil {
		return downloadPin{}, err
	}
	var p downloadPin
	if err = json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	if p.ID == "" || (id != "" && p.ID != id) || FilePath(filepath.Dir(private), p.ID) != private {
		return p, errors.New("invalid download record")
	}
	if p.Filename != "" && (filepath.Base(p.Filename) != p.Filename || strings.ContainsRune(p.Filename, '\\') || filepath.Ext(p.Filename) != ".mp3") {
		return p, errors.New("invalid download filename")
	}
	return p, nil
}

func (p downloadPin) store(private string) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return writeAtomic(pinPath(private), b)
}

func hasAudio(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode().IsRegular() && st.Size() >= 1024 && st.Size() <= MaxTrackBytes
}

func PlayablePath(shared, dir, id string) string {
	private := FilePath(dir, id)
	if p, err := readPin(private, id); err == nil && p.Filename != "" {
		path := filepath.Join(shared, p.Filename)
		if hasAudio(path) {
			return path
		}
	}
	return private
}

func songName(s Song) string {
	title := strings.TrimSpace(s.Title)
	if title == "" {
		title = "Untitled"
	}
	if artist := strings.TrimSpace(s.Artist); artist != "" {
		title = artist + " - " + title
	}
	var b strings.Builder
	for _, r := range title {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			r = '-'
		}
		if b.Len() >= 176 {
			break
		}
		b.WriteRune(r)
	}
	name := strings.Trim(b.String(), " .")
	if name == "" {
		return "Untitled"
	}
	first, _, _ := strings.Cut(strings.ToUpper(name), ".")
	switch strings.TrimSpace(first) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return "_" + name
	}
	return name
}

// Reserve the name exclusively on the destination filesystem (including FAT).
// The move then replaces only our own empty reservation, even on name collisions.
func reserveMP3(shared, private string, s Song) (string, error) {
	if err := os.MkdirAll(shared, 0755); err != nil {
		return "", err
	}
	// A stable suffix keeps a deleted track’s record from claiming a later, identically named song.
	key := sha256.Sum256([]byte(filepath.Base(filepath.Dir(private)) + "\x00" + s.ID))
	base := fmt.Sprintf("%s [%x]", songName(s), key[:6])
	for i := 1; ; i++ {
		name := base + ".mp3"
		if i > 1 {
			name = fmt.Sprintf("%s (%d).mp3", base, i)
		}
		f, err := os.OpenFile(filepath.Join(shared, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		return name, f.Close()
	}
}

func publish(private, shared string, p downloadPin) error {
	if p.Filename != "" && hasAudio(filepath.Join(shared, p.Filename)) {
		if !p.Pending {
			return nil
		}
		p.Pending = false
		return p.store(private)
	}
	if !hasAudio(private) {
		return os.ErrNotExist
	}
	// After a power loss, continue a journaled move into its original reservation.
	dest := filepath.Join(shared, p.Filename)
	st, err := os.Lstat(dest)
	if !p.Pending || p.Filename == "" || err != nil || !st.Mode().IsRegular() || st.Size() != 0 {
		name, err := reserveMP3(shared, private, p.Song)
		if err != nil {
			return err
		}
		dest = filepath.Join(shared, name)
		p.Filename, p.Pending = name, true
		if err = p.store(private); err != nil {
			_ = os.Remove(dest)
			return err
		}
	}
	if err = os.Rename(private, dest); err != nil {
		return fmt.Errorf("moving download into Media/Music: %w", err)
	}
	p.Pending = false
	return p.store(private)
}

func (c *Client) SaveSong(ctx context.Context, root, shared string, s Song) error {
	storageMu.Lock()
	defer storageMu.Unlock()
	private := FilePath(c.MusicDir(root), s.ID)
	p, err := readPin(private, s.ID)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	p.Song = s
	if p.Filename == "" || !hasAudio(filepath.Join(shared, p.Filename)) {
		if _, err = c.CacheSong(ctx, s, c.MusicDir(root)); err != nil {
			return err
		}
	}
	return publish(private, shared, p)
}

type SavedSong struct {
	Song Song
	Dir  string
	Path string
}

func (s SavedSong) CoverPath() string {
	id := s.Song.CoverArt
	if id == "" {
		id = s.Song.AlbumID
	}
	if id == "" {
		id = s.Song.ID
	}
	return artPath(s.Dir, id)
}

func visitPins(root string, visit func(string, downloadPin) error) error {
	paths, err := filepath.Glob(filepath.Join(root, "*", "saved", "*.json"))
	if err != nil {
		return err
	}
	var first error
	for _, path := range paths {
		dir := filepath.Dir(filepath.Dir(path))
		if !hexName(filepath.Base(dir), 64) {
			continue
		}
		private := filepath.Join(dir, strings.TrimSuffix(filepath.Base(path), ".json")+".mp3")
		p, err := readPin(private, "")
		if err == nil {
			err = visit(private, p)
		}
		if err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Only old explicitly saved tracks and interrupted moves need migration.
func MigrateDownloads(root, shared string) error {
	storageMu.Lock()
	defer storageMu.Unlock()
	return visitPins(root, func(private string, p downloadPin) error {
		if p.Filename != "" && !p.Pending {
			return nil
		}
		if !p.Pending && !hasAudio(private) {
			return nil
		}
		return publish(private, shared, p)
	})
}

func DownloadedSongs(root, shared string) ([]SavedSong, error) {
	var out []SavedSong
	err := visitPins(root, func(private string, p downloadPin) error {
		path := private
		if p.Filename != "" {
			public := filepath.Join(shared, p.Filename)
			if hasAudio(public) {
				path = public
			} else if !p.Pending {
				return nil
			}
		}
		if hasAudio(path) {
			out = append(out, SavedSong{p.Song, filepath.Dir(private), path})
		}
		return nil
	})
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Song.Title) < strings.ToLower(out[j].Song.Title) })
	return out, err
}

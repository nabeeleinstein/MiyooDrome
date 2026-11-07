package navcore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const DefaultCacheLimit int64 = 256 << 20

var storageMu sync.Mutex

func hexName(name string, n int) bool {
	if len(name) != n {
		return false
	}
	_, err := hex.DecodeString(name)
	return err == nil
}

// Recognize only names produced by MiyooDrome 0.2.x or this version.
func ManagedTrack(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 1 && (len(parts) != 2 || !hexName(parts[0], 64)) {
		return false
	}
	name := filepath.Base(path)
	return filepath.Ext(name) == ".mp3" && hexName(strings.TrimSuffix(name, ".mp3"), 32)
}

func pinPath(path string) string {
	return filepath.Join(filepath.Dir(path), "saved", strings.TrimSuffix(filepath.Base(path), ".mp3")+".json")
}

func TemporaryTrack(root, path string) bool {
	if !ManagedTrack(root, path) {
		return false
	}
	_, err := os.Stat(pinPath(path))
	return os.IsNotExist(err)
}

type cacheFile struct {
	path string
	size int64
	used time.Time
}

func cacheFiles(root string) ([]cacheFile, int64, error) {
	var files []cacheFile
	var total int64
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	add := func(dir string, e os.DirEntry) error {
		path := filepath.Join(dir, e.Name())
		if !e.Type().IsRegular() || !TemporaryTrack(root, path) {
			return nil
		}
		st, err := e.Info()
		if err != nil {
			return err
		}
		total += st.Size()
		files = append(files, cacheFile{path, st.Size(), st.ModTime()})
		return nil
	}
	for _, e := range entries {
		if e.IsDir() && hexName(e.Name(), 64) {
			dir := filepath.Join(root, e.Name())
			children, err := os.ReadDir(dir)
			if err != nil {
				return nil, 0, err
			}
			for _, child := range children {
				if err := add(dir, child); err != nil {
					return nil, 0, err
				}
			}
		} else if err := add(root, e); err != nil {
			return nil, 0, err
		}
	}
	return files, total, nil
}

func CacheUsage(root string) (int64, error) {
	storageMu.Lock()
	defer storageMu.Unlock()
	_, total, err := cacheFiles(root)
	return total, err
}

// The active song remains available for seek and resume, including during a manual clear.
func TrimCache(root string, limit int64, active string) (int64, error) {
	storageMu.Lock()
	defer storageMu.Unlock()
	return trimCache(root, limit, active)
}

func trimCache(root string, limit int64, active string) (int64, error) {
	files, total, err := cacheFiles(root)
	if err != nil {
		return 0, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].used.Before(files[j].used) })
	var removed int64
	for _, file := range files {
		if total <= limit {
			break
		}
		if file.path == active {
			continue
		}
		if err := os.Remove(file.path); err != nil && !os.IsNotExist(err) {
			return removed, err
		}
		total -= file.size
		removed += file.size
	}
	return removed, nil
}

func (c *Client) CacheForPlayback(ctx context.Context, root string, s Song, limit int64, active string) (string, error) {
	storageMu.Lock()
	defer storageMu.Unlock()
	dir := c.MusicDir(root)
	path, err := c.cacheSong(ctx, s, dir, func(reserve int64) error {
		_, err := trimCache(root, limit-reserve, active)
		return err
	})
	if err == nil {
		now := time.Now()
		_ = os.Chtimes(path, now, now)
	}
	return path, err
}

func (c *Client) SaveSong(ctx context.Context, root string, s Song) error {
	storageMu.Lock()
	defer storageMu.Unlock()
	path, err := c.CacheSong(ctx, s, c.MusicDir(root))
	if err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return writeAtomic(pinPath(path), data)
}

type SavedSong struct {
	Song Song
	Dir  string
}

func DownloadedSongs(root string) ([]SavedSong, error) {
	paths, err := filepath.Glob(filepath.Join(root, "*", "saved", "*.json"))
	if err != nil {
		return nil, err
	}
	var songs []SavedSong
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var s Song
		if json.Unmarshal(b, &s) != nil || s.ID == "" {
			continue
		}
		dir := filepath.Dir(filepath.Dir(p))
		if pinPath(FilePath(dir, s.ID)) != p {
			continue
		}
		if _, err := os.Stat(FilePath(dir, s.ID)); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("reading downloaded song: %w", err)
		}
		songs = append(songs, SavedSong{s, dir})
	}
	sort.SliceStable(songs, func(i, j int) bool {
		return strings.ToLower(songs[i].Song.Title) < strings.ToLower(songs[j].Song.Title)
	})
	return songs, nil
}

// ReadSavedSong supplies server metadata and local artwork to the offline library scanner.
func ReadSavedSong(root, path string) (Song, string, bool) {
	if !ManagedTrack(root, path) {
		return Song{}, "", false
	}
	b, err := os.ReadFile(pinPath(path))
	if err != nil {
		return Song{}, "", false
	}
	var s Song
	if json.Unmarshal(b, &s) != nil || s.ID == "" || FilePath(filepath.Dir(path), s.ID) != path {
		return Song{}, "", false
	}
	id := s.CoverArt
	if id == "" {
		id = s.AlbumID
	}
	if id == "" {
		id = s.ID
	}
	return s, artPath(filepath.Dir(path), id), true
}

// Run before any transfers on startup, so a hard power-off cannot strand partial MP3s.
func CleanupInterrupted(root string) error {
	dirs, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	locations := []string{root}
	for _, d := range dirs {
		if d.IsDir() && hexName(d.Name(), 64) {
			locations = append(locations, filepath.Join(root, d.Name()))
		}
	}
	for _, dir := range locations {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Type().IsRegular() && strings.HasPrefix(e.Name(), ".navidrome-") && strings.HasSuffix(e.Name(), ".part") {
				if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

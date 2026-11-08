package navcore

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Album manifests and their removal journals must never overlap a new album save.
var albumMu sync.Mutex

func albumPath(dir, id string) string {
	return filepath.Join(dir, "albums", fmt.Sprintf("%x.json", sha256.Sum256([]byte(id))))
}

func readAlbum(path string) (SavedAlbum, error) {
	var a SavedAlbum
	b, err := os.ReadFile(path)
	if err != nil {
		return a, err
	}
	if err = json.Unmarshal(b, &a); err != nil {
		return a, err
	}
	a.Dir = filepath.Dir(filepath.Dir(path))
	if a.Album.ID == "" || albumPath(a.Dir, a.Album.ID) != strings.TrimSuffix(path, ".deleting") {
		return a, errors.New("invalid album record")
	}
	return a, nil
}

func removeIfPresent(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func DeleteAlbum(root, shared, dir, id string) error {
	albumMu.Lock()
	defer albumMu.Unlock()
	storageMu.Lock()
	defer storageMu.Unlock()
	if id == "" || filepath.Dir(dir) != filepath.Clean(root) || !hexName(filepath.Base(dir), 64) {
		return errors.New("invalid download folder")
	}
	path := albumPath(dir, id)
	journal := path + ".deleting"
	if _, err := os.Stat(journal); os.IsNotExist(err) {
		if _, err := readAlbum(path); err != nil {
			return err
		}
		if err := os.Rename(path, journal); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return finishAlbumRemoval(shared, journal)
}

// A journal survives a power cut. Remove the pin last so its public filename
// remains available until both the public copy and any legacy private copy are gone.
func finishAlbumRemoval(shared, journal string) error {
	a, err := readAlbum(journal)
	if err != nil {
		return err
	}
	keep := map[string]bool{}
	others, err := filepath.Glob(filepath.Join(a.Dir, "albums", "*.json"))
	if err != nil {
		return err
	}
	for _, path := range others {
		other, err := readAlbum(path)
		if err != nil {
			return err
		}
		for _, s := range other.Songs {
			keep[s.ID] = true
		}
	}
	for _, song := range a.Songs {
		if keep[song.ID] {
			continue
		}
		private := FilePath(a.Dir, song.ID)
		pin, err := readPin(private, song.ID)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if pin.Standalone {
			continue
		}
		if err := removeSongFiles(shared, private, pin); err != nil {
			return err
		}
	}
	return removeIfPresent(journal)
}

func ResumeAlbumRemovals(root, shared string) error {
	albumMu.Lock()
	defer albumMu.Unlock()
	storageMu.Lock()
	defer storageMu.Unlock()
	paths, err := filepath.Glob(filepath.Join(root, "*", "albums", "*.json.deleting"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		if !hexName(filepath.Base(filepath.Dir(filepath.Dir(path))), 64) {
			continue
		}
		if err := finishAlbumRemoval(shared, path); err != nil {
			return err
		}
	}
	return nil
}

// DeleteSong removes the requested local download even when it came from an album.
// The album browser filters missing tracks while retaining the album removal action.
func DeleteSong(root, shared, dir, id string) error {
	albumMu.Lock()
	defer albumMu.Unlock()
	storageMu.Lock()
	defer storageMu.Unlock()
	if id == "" || filepath.Dir(dir) != filepath.Clean(root) || !hexName(filepath.Base(dir), 64) {
		return errors.New("invalid download folder")
	}
	private := FilePath(dir, id)
	pin, err := readPin(private, id)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	pin.Deleting = true
	if err := pin.store(private); err != nil {
		return err
	}
	return removeSongFiles(shared, private, pin)
}

func validateDownloadName(private string, pin downloadPin) error {
	if pin.Filename != "" {
		key := sha256.Sum256([]byte(filepath.Base(filepath.Dir(private)) + "\x00" + pin.ID))
		suffix := fmt.Sprintf(" [%x]", key[:6])
		stem := strings.TrimSuffix(pin.Filename, ".mp3")
		if pos := strings.LastIndex(stem, " ("); pos >= 0 && strings.HasSuffix(stem, ")") {
			stem = stem[:pos]
		}
		if !strings.HasSuffix(stem, suffix) {
			return errors.New("download filename does not match its record")
		}
	}
	return nil
}

func removeSongFiles(shared, private string, pin downloadPin) error {
	if err := validateDownloadName(private, pin); err != nil {
		return err
	}
	if pin.Filename != "" {
		if err := removeIfPresent(filepath.Join(shared, pin.Filename)); err != nil {
			return err
		}
	}
	if err := removeIfPresent(private); err != nil {
		return err
	}
	return removeIfPresent(pinPath(private))
}

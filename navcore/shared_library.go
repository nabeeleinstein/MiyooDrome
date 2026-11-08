package navcore

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// MergeLegacyDownloads runs before playback starts. Consume each old record only
// after its audio, cover and new record are durable. Loose files are never adopted.
func MergeLegacyDownloads(oldRoot, oldPublic, root, public string) error {
	if oldRoot == root || oldPublic == public {
		return errors.New("migration requires distinct libraries")
	}
	if err := ResumeAlbumRemovals(oldRoot, oldPublic); err != nil {
		return err
	}
	if err := MigrateDownloads(oldRoot, oldPublic); err != nil {
		return err
	}
	albumMu.Lock()
	defer albumMu.Unlock()
	storageMu.Lock()
	defer storageMu.Unlock()
	if err := visitPins(oldRoot, func(oldPrivate string, old downloadPin) error {
		dir := filepath.Join(root, filepath.Base(filepath.Dir(oldPrivate)))
		private := FilePath(dir, old.ID)
		if old.Filename == "" {
			return nil
		}
		if err := validateDownloadName(oldPrivate, old); err != nil {
			return err
		}
		src := filepath.Join(oldPublic, old.Filename)
		p, err := readPin(private, old.ID)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := validateDownloadName(private, p); err != nil {
			return err
		}
		if p.Deleting {
			return errors.New("finish pending download removal before migration")
		}
		dst := filepath.Join(public, p.Filename)
		// A missing old file may already have been moved before a power cut.
		if !hasAudio(src) && !hasAudio(dst) {
			return nil
		}
		if p.ID == "" {
			p.Song = old.Song
		}
		p.Standalone = p.Standalone || old.Standalone
		if err := copySavedCover(SavedSong{Song: old.Song, Dir: filepath.Dir(oldPrivate)}.CoverPath(), SavedSong{Song: p.Song, Dir: dir}.CoverPath()); err != nil {
			return err
		}
		if !hasAudio(dst) {
			info, statErr := os.Lstat(dst)
			if !p.Pending || statErr != nil || !info.Mode().IsRegular() || info.Size() != 0 {
				name, err := reserveMP3(public, private, old.Song)
				if err != nil {
					return err
				}
				p.Filename, p.Pending = name, true
				dst = filepath.Join(public, name)
				if err := p.store(private); err != nil {
					_ = os.Remove(dst)
					return err
				}
			}
			if err := os.Rename(src, dst); err != nil {
				return err
			}
		}
		p.Pending = false
		if err := p.store(private); err != nil {
			return err
		}
		if hasAudio(src) {
			same, err := identicalAudio(src, dst)
			if err != nil {
				return err
			}
			if same {
				if err := os.Remove(src); err != nil {
					return err
				}
			}
		}
		return removeIfPresent(pinPath(oldPrivate))
	}); err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(oldRoot, "*", "albums", "*.json"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		old, err := readAlbum(path)
		if err != nil {
			return err
		}
		account := filepath.Base(old.Dir)
		if !hexName(account, 64) {
			continue
		}
		dir := filepath.Join(root, account)
		dst := albumPath(dir, old.Album.ID)
		a, err := readAlbum(dst)
		if os.IsNotExist(err) {
			a = old
		} else if err != nil {
			return err
		} else {
			seen := make(map[string]bool)
			for _, s := range a.Songs {
				seen[s.ID] = true
			}
			for _, s := range old.Songs {
				if !seen[s.ID] {
					a.Songs = append(a.Songs, s)
					seen[s.ID] = true
				}
			}
		}
		if old.Album.CoverArt != "" && a.Album.CoverArt != "" {
			if err := copySavedCover(artPath(old.Dir, old.Album.CoverArt), artPath(dir, a.Album.CoverArt)); err != nil {
				return err
			}
		}
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}
		if err := writeAtomic(dst, data); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

func copySavedCover(src, dst string) error {
	if st, err := os.Stat(dst); err == nil && st.Size() > 0 && st.Size() <= 2<<20 {
		if b, err := os.ReadFile(dst); err == nil {
			if _, err := decodeCover(b); err == nil {
				return nil
			}
		}
	}
	st, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Size() > 2<<20 {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if _, err := decodeCover(b); err != nil {
		return nil
	}
	return writeAtomic(dst, b)
}

func identicalAudio(a, b string) (bool, error) {
	digest := func(path string) ([]byte, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		h := sha256.New()
		_, err = io.Copy(h, f)
		return h.Sum(nil), err
	}
	x, err := digest(a)
	if err != nil {
		return false, err
	}
	y, err := digest(b)
	return bytes.Equal(x, y), err
}

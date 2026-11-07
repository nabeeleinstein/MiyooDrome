package navcore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

func (c *Client) MusicDir(root string) string {
	hash := sha256.Sum256([]byte(strings.TrimRight(c.Config.Server, "/") + "\x00" + c.Config.Username))
	return filepath.Join(root, fmt.Sprintf("%x", hash))
}

func artPath(dir, id string) string {
	return filepath.Join(dir, "covers", fmt.Sprintf("%x.img", sha256.Sum256([]byte(id))))
}

func decodeCover(data []byte) (image.Image, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 1024 || cfg.Height > 1024 {
		return nil, errors.New("invalid cover image")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// Cover returns a bounded, decoded image. A nil client reads the offline cache only.
func Cover(ctx context.Context, c *Client, dir, id string) (image.Image, error) {
	if id == "" {
		return nil, errors.New("no cover ID")
	}
	path := artPath(dir, id)
	if info, err := os.Stat(path); err == nil && info.Size() <= 2<<20 {
		if data, err := os.ReadFile(path); err == nil {
			if img, err := decodeCover(data); err == nil {
				return img, nil
			}
		}
	}
	if c == nil {
		return nil, os.ErrNotExist
	}
	addr, err := c.Endpoint("getCoverArt", url.Values{"id": {id}, "size": {"200"}})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}
	r, err := c.HTTP.Do(req)
	if err != nil {
		return nil, errors.New("cover unavailable")
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("cover HTTP %d", r.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, (2<<20)+1))
	if err != nil || len(data) > 2<<20 {
		return nil, errors.New("invalid cover response")
	}
	img, err := decodeCover(data)
	if err != nil {
		return nil, err
	}
	// A full or read-only card should still show the image fetched in this session.
	_ = writeAtomic(path, data)
	return img, nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp.Name(), path)
}

type SavedAlbum struct {
	Album Album  `json:"album"`
	Songs []Song `json:"songs"`
	Dir   string `json:"-"`
}

func (c *Client) DownloadAlbum(ctx context.Context, root, shared string, a Album, songs []Song, progress func(int, string)) error {
	albumMu.Lock()
	defer albumMu.Unlock()
	if len(songs) == 0 {
		return errors.New("album contains no tracks")
	}
	dir := c.MusicDir(root)
	if _, err := os.Stat(albumPath(dir, a.ID) + ".deleting"); err == nil {
		return errors.New("finish removing this album before downloading it again")
	} else if !os.IsNotExist(err) {
		return err
	}
	coverID := a.CoverArt
	if coverID == "" {
		coverID = songs[0].CoverArt
	}
	if coverID == "" {
		coverID = a.ID
	}
	a.CoverArt = coverID
	songs = append([]Song(nil), songs...)
	for i := range songs {
		songs[i].CoverArt = coverID
	}
	for i, s := range songs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if progress != nil {
			progress(i, s.Title)
		}
		trackCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := c.saveSong(trackCtx, root, shared, s, false)
		cancel()
		if err != nil {
			return fmt.Errorf("track %d of %d: %w", i+1, len(songs), err)
		}
	}
	coverCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	_, _ = Cover(coverCtx, c, dir, coverID)
	cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(SavedAlbum{Album: a, Songs: songs})
	if err != nil {
		return err
	}
	return writeAtomic(albumPath(dir, a.ID), data)
}

func DownloadedAlbums(root, shared string) ([]SavedAlbum, error) {
	paths, err := filepath.Glob(filepath.Join(root, "*", "albums", "*.json"))
	if err != nil {
		return nil, err
	}
	var albums []SavedAlbum
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var a SavedAlbum
		if json.Unmarshal(data, &a) != nil || a.Album.ID == "" || len(a.Songs) == 0 {
			continue
		}
		a.Dir = filepath.Dir(filepath.Dir(p))
		complete := true
		for _, s := range a.Songs {
			private := FilePath(a.Dir, s.ID)
			path := PlayablePath(shared, a.Dir, s.ID)
			if path == private && (TemporaryTrack(root, private) || !hasAudio(path)) {
				complete = false
				break
			}
		}
		if complete {
			albums = append(albums, a)
		}
	}
	sort.SliceStable(albums, func(i, j int) bool {
		return strings.ToLower(albums[i].Album.Name) < strings.ToLower(albums[j].Album.Name)
	})
	return albums, nil
}

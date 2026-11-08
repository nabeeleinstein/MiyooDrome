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
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/dhowden/tag"
	xdraw "golang.org/x/image/draw"
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

// Cover reads the cached artwork first. A nil client never contacts a server.
func Cover(ctx context.Context, c *Client, dir, id string) (image.Image, error) {
	return ResolveCover(ctx, c, dir, id, "")
}

// ResolveCover stores recovered artwork under the same ID the player and scanner read.
func ResolveCover(ctx context.Context, c *Client, dir, id, audioPath string, alternatives ...string) (image.Image, error) {
	if id == "" {
		return nil, errors.New("no cover ID")
	}
	ids := []string{id}
	for _, alt := range alternatives {
		if alt != "" && !slices.Contains(ids, alt) {
			ids = append(ids, alt)
		}
	}
	path := artPath(dir, id)
	for _, candidate := range ids {
		p := artPath(dir, candidate)
		if info, err := os.Stat(p); err == nil && info.Size() <= 2<<20 {
			if data, err := os.ReadFile(p); err == nil {
				if img, err := decodeCover(data); err == nil {
					if p != path {
						return img, writeAtomic(path, data)
					}
					return img, nil
				}
			}
		}
	}
	if audioPath != "" {
		if data, img := embeddedCover(audioPath); img != nil {
			return img, writeAtomic(path, data)
		}
	}
	if c == nil {
		return nil, os.ErrNotExist
	}
	var lastErr error
	for _, candidate := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := c.fetchCover(ctx, candidate)
		if err == nil {
			var img image.Image
			data, img, err = prepareCover(data)
			if err == nil {
				return img, writeAtomic(path, data)
			}
		}
		lastErr = err
	}
	return nil, lastErr
}

func embeddedCover(path string) ([]byte, image.Image) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()
	// Navidrome downloads are MP3s. Only inspect leading metadata on the handheld.
	meta, err := tag.ReadFrom(io.NewSectionReader(f, 0, 4<<20))
	if err != nil {
		return nil, nil
	}
	pic := meta.Picture()
	if pic == nil || len(pic.Data) > 4<<20 {
		return nil, nil
	}
	data, img, _ := prepareCover(pic.Data)
	return data, img
}

// Some servers return the original image despite the requested thumbnail size.
func prepareCover(data []byte) ([]byte, image.Image, error) {
	if len(data) <= 2<<20 {
		if img, err := decodeCover(data); err == nil {
			return data, img, nil
		}
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 4_000_000 {
		return nil, nil, errors.New("invalid cover image")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	thumb := image.NewRGBA(image.Rect(0, 0, 200, 200))
	xdraw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, thumb, nil); err != nil {
		return nil, nil, err
	}
	return encoded.Bytes(), thumb, nil
}

func (c *Client) fetchCover(ctx context.Context, id string) ([]byte, error) {
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
	data, err := io.ReadAll(io.LimitReader(r.Body, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return nil, errors.New("invalid cover response")
	}
	return data, nil
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
	_, err := c.DownloadAlbumWithArtwork(ctx, root, shared, a, songs, progress)
	return err
}

func (c *Client) DownloadAlbumWithArtwork(ctx context.Context, root, shared string, a Album, songs []Song, progress func(int, string)) (bool, error) {
	albumMu.Lock()
	defer albumMu.Unlock()
	if len(songs) == 0 {
		return false, errors.New("album contains no tracks")
	}
	dir := c.MusicDir(root)
	if _, err := os.Stat(albumPath(dir, a.ID) + ".deleting"); err == nil {
		return false, errors.New("finish removing this album before downloading it again")
	} else if !os.IsNotExist(err) {
		return false, err
	}
	coverID := a.CoverArt
	if coverID == "" {
		coverID = songs[0].CoverArt
	}
	if coverID == "" {
		coverID = a.ID
	}
	a.CoverArt = coverID
	// Keep track cover IDs for servers that cannot resolve the album artwork ID.
	fallbacks := []string{a.ID, songs[0].ID}
	for _, song := range songs {
		if song.CoverArt != "" && !slices.Contains(fallbacks, song.CoverArt) {
			fallbacks = append(fallbacks, song.CoverArt)
		}
	}
	songs = append([]Song(nil), songs...)
	for i := range songs {
		songs[i].CoverArt = coverID
	}
	for i, s := range songs {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if progress != nil {
			progress(i, s.Title)
		}
		trackCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := c.saveSong(trackCtx, root, shared, s, false)
		cancel()
		if err != nil {
			return false, fmt.Errorf("track %d of %d: %w", i+1, len(songs), err)
		}
	}
	coverCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	_, artErr := ResolveCover(coverCtx, c, dir, coverID, PlayablePath(shared, dir, songs[0].ID), fallbacks...)
	cancel()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	data, err := json.Marshal(SavedAlbum{Album: a, Songs: songs})
	if err != nil {
		return false, err
	}
	return artErr == nil, writeAtomic(albumPath(dir, a.ID), data)
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
		available := a.Songs[:0]
		for _, s := range a.Songs {
			private := FilePath(a.Dir, s.ID)
			path := PlayablePath(shared, a.Dir, s.ID)
			if path == private && (TemporaryTrack(root, private) || !hasAudio(path)) {
				continue
			}
			available = append(available, s)
		}
		a.Songs = available
		if len(a.Songs) > 0 {
			albums = append(albums, a)
		}
	}
	sort.SliceStable(albums, func(i, j int) bool {
		return strings.ToLower(albums[i].Album.Name) < strings.ToLower(albums[j].Album.Name)
	})
	return albums, nil
}

package navcore

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedDownloadMigration(t *testing.T) {
	c, err := New(cfg("https://example.org"))
	if err != nil {
		t.Fatal(err)
	}
	old, from, root, to := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	songs := []Song{{ID: "one", AlbumID: "record", CoverArt: "art"}, {ID: "two", AlbumID: "record", CoverArt: "art"}, {ID: "three", AlbumID: "record"}}
	seed := func(root, shared string, s Song, b byte, solo bool) string {
		t.Helper()
		dir := c.MusicDir(root)
		if err := writeAtomic(FilePath(dir, s.ID), bytes.Repeat([]byte{b}, 2048)); err != nil {
			t.Fatal(err)
		}
		if err := c.saveSong(context.Background(), root, shared, s, solo); err != nil {
			t.Fatal(err)
		}
		return PlayablePath(shared, dir, s.ID)
	}
	var art bytes.Buffer
	if err := png.Encode(&art, image.NewGray(image.Rect(0, 0, 3, 3))); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(artPath(c.MusicDir(old), "art"), art.Bytes()); err != nil {
		t.Fatal(err)
	}
	for _, s := range songs {
		seed(old, from, s, 1, s.ID == "one")
	}
	seed(root, to, songs[0], 1, false)
	seed(root, to, songs[2], 2, false)
	oldVariant := PlayablePath(from, c.MusicDir(old), songs[2].ID)
	loose := filepath.Join(from, "manual.mp3")
	if err := os.WriteFile(loose, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	// Simulate a power cut after the MP3 has moved but before the new pin commits.
	dir := c.MusicDir(root)
	p := downloadPin{Song: songs[1], Pending: true}
	private := FilePath(dir, p.ID)
	p.Filename, err = reserveMP3(to, private, p.Song)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.store(private); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(PlayablePath(from, c.MusicDir(old), p.ID), filepath.Join(to, p.Filename)); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		root  string
		songs []Song
	}{{old, songs}, {root, songs[:1]}} {
		data, _ := json.Marshal(SavedAlbum{Album: Album{ID: "record", CoverArt: "art"}, Songs: entry.songs})
		if err := writeAtomic(albumPath(c.MusicDir(entry.root), "record"), data); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := MergeLegacyDownloads(old, from, root, to); err != nil {
			t.Fatal(err)
		}
	}
	saved, err := DownloadedSongs(root, to)
	if err != nil || len(saved) != 3 {
		t.Fatal(saved, err)
	}
	albums, err := DownloadedAlbums(root, to)
	if err != nil || len(albums) != 1 || len(albums[0].Songs) != 3 {
		t.Fatal(albums, err)
	}
	if _, err := Cover(context.Background(), nil, dir, "art"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{loose, oldVariant} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	if err := DeleteAlbum(root, to, dir, "record"); err != nil {
		t.Fatal(err)
	}
	if err := MergeLegacyDownloads(old, from, root, to); err != nil {
		t.Fatal(err)
	}
	saved, err = DownloadedSongs(root, to)
	if err != nil || len(saved) != 1 || saved[0].Song.ID != "one" {
		t.Fatal("standalone protection / deleted album resurrected", saved, err)
	}
}

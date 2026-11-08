package navcore

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAlbumArtSurvivesOfflineRestart(t *testing.T) {
	// Simulate a server that ignores the requested thumbnail size and only
	// recognizes the track cover ID included on the second track.
	var picture bytes.Buffer
	art := image.NewRGBA(image.Rect(0, 0, 1200, 1200))
	if err := png.Encode(&picture, art); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/stream.view" {
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(append(MakeID3Tag(Song{}), bytes.Repeat([]byte{0}, 2048)...))
			return
		}
		if r.URL.Path == "/rest/getCoverArt.view" && r.URL.Query().Get("id") == "track-cover" {
			w.Write(picture.Bytes())
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	c, err := New(cfg(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	root, shared := t.TempDir(), t.TempDir()
	album := Album{ID: "album", CoverArt: "album-cover"}
	songs := []Song{{ID: "first", AlbumID: "album"}, {ID: "second", AlbumID: "album", CoverArt: "track-cover"}}
	ok, err := c.DownloadAlbumWithArtwork(context.Background(), root, shared, album, songs, nil)
	if err != nil || !ok {
		t.Fatalf("art saved %v, err %v", ok, err)
	}
	server.Close()
	saved, err := DownloadedSongs(root, shared)
	if err != nil || len(saved) != 2 {
		t.Fatal("cannot re-open album", err)
	}
	for _, s := range saved {
		img, err := Cover(context.Background(), nil, s.Dir, s.Song.CoverArt)
		if err != nil || img.Bounds().Dx() != 200 {
			t.Fatalf("offline artwork for %s: %v", s.Song.ID, err)
		}
	}
}

func TestEmbeddedArtRepairAndWriteFailure(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	want := color.RGBA{30, 45, 139, 255}
	img.SetRGBA(0, 0, want)
	var picture bytes.Buffer
	if err := png.Encode(&picture, img); err != nil {
		t.Fatal(err)
	}
	payload := append([]byte("\x00image/png\x00\x03\x00"), picture.Bytes()...)
	frame := append([]byte("APIC"), syncsafe(len(payload))...)
	frame = append(frame, 0, 0)
	frame = append(frame, payload...)
	audio := append([]byte("ID3\x04\x00\x00"), syncsafe(len(frame))...)
	path := filepath.Join(t.TempDir(), "saved.mp3")
	if err := os.WriteFile(path, append(audio, frame...), 0600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	got, err := ResolveCover(context.Background(), nil, dir, "front", path)
	if err != nil || color.RGBAModel.Convert(got.At(0, 0)) != want {
		t.Fatal("embedded artwork", err)
	}
	if _, err := Cover(context.Background(), nil, dir, "front"); err != nil {
		t.Fatal("offline cache", err)
	}
	blocked := t.TempDir()
	if err := os.WriteFile(filepath.Join(blocked, "covers"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if live, err := ResolveCover(context.Background(), nil, blocked, "front", path); live == nil || err == nil {
		t.Fatal("cache write failure should retain live picture and surface the error", err)
	}
}

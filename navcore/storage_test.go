package navcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRetentionAcrossAccountsAndOldInstalls(t *testing.T) {
	root := t.TempDir()
	account := filepath.Join(root, strings.Repeat("a", 64))
	os.MkdirAll(account, 0700)
	other := filepath.Join(root, strings.Repeat("b", 64))
	os.MkdirAll(other, 0700)
	add := func(dir, id string, age int64) string {
		p := FilePath(dir, id)
		if err := os.WriteFile(p, bytes.Repeat([]byte("a"), 2048), 0600); err != nil {
			t.Fatal(err)
		}
		when := time.Unix(age, 0)
		if err := os.Chtimes(p, when, when); err != nil {
			t.Fatal(err)
		}
		return p
	}
	oldest := add(root, "legacy", 1)
	active := add(account, "playing", 2)
	recent := add(other, "recent", 3)
	kept := add(account, "download", 1)
	if err := writeAtomic(pinPath(kept), []byte("{}")); err != nil {
		t.Fatal(err)
	}
	personal := filepath.Join(root, "manually copied.mp3")
	os.WriteFile(personal, []byte("mine"), 0600)
	removed, err := TrimCache(root, 4096, active)
	if err != nil || removed != 2048 {
		t.Fatal(removed, err)
	}
	if _, err := os.Stat(oldest); !os.IsNotExist(err) {
		t.Fatal("old track retained")
	}
	if _, err := os.Stat(recent); err != nil {
		t.Fatal("newer track evicted")
	}
	removed, err = TrimCache(root, 0, active)
	if err != nil || removed != 2048 {
		t.Fatal(removed, err)
	}
	for _, p := range []string{active, kept, personal} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal("removed protected file", p)
		}
	}
}

func TestAlbumRetryPinsOnlyCompletedSongs(t *testing.T) {
	attempts := map[string]int{}
	bad := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if r.URL.Path != "/rest/stream.view" {
			http.NotFound(w, r)
			return
		}
		attempts[id]++
		if id == "b" && bad {
			http.Error(w, "failure", 500)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(append([]byte("ID3"), bytes.Repeat([]byte{1}, 2048)...))
	}))
	defer server.Close()
	c, err := New(cfg(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	root, shared := t.TempDir(), t.TempDir()
	a := Album{ID: "album", Name: "Album"}
	songs := []Song{{ID: "a", Title: "First"}, {ID: "b", Title: "Second"}}
	ctx := context.Background()
	if _, err = c.CacheForPlayback(ctx, root, shared, songs[0], DefaultCacheLimit, ""); err != nil {
		t.Fatal(err)
	}
	if err = c.DownloadAlbum(ctx, root, shared, a, songs, nil); err == nil {
		t.Fatal("partial album was accepted")
	}
	albums, _ := DownloadedAlbums(root, shared)
	if len(albums) != 0 {
		t.Fatal("partial album listed")
	}
	saved, _ := DownloadedSongs(root, shared)
	if len(saved) != 1 {
		t.Fatal("completed song missing")
	}
	if _, err = TrimCache(root, 0, ""); err != nil {
		t.Fatal(err)
	}
	bad = false
	if err = c.DownloadAlbum(ctx, root, shared, a, songs, nil); err != nil {
		t.Fatal(err)
	}
	saved, _ = DownloadedSongs(root, shared)
	albums, _ = DownloadedAlbums(root, shared)
	if len(saved) != 2 || len(albums) != 1 || attempts["a"] != 1 || attempts["b"] != 2 {
		t.Fatal("retry fetched a completed track", attempts)
	}
}

func TestPlaybackEnforcesBudgetOnEveryMiss(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := append([]byte("ID3"), bytes.Repeat([]byte{1}, 3000)...)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.Write(data)
	}))
	defer server.Close()
	c, _ := New(cfg(server.URL))
	root, shared := t.TempDir(), t.TempDir()
	for _, id := range []string{"a", "b", "c", "d"} {
		if _, err := c.CacheForPlayback(context.Background(), root, shared, Song{ID: id}, 7000, ""); err != nil {
			t.Fatal(err)
		}
		used, err := CacheUsage(root)
		if err != nil || used > 7000 {
			t.Fatal("budget exceeded", used, err)
		}
	}
}

func TestPublicMP3Migration(t *testing.T) {
	root, shared := t.TempDir(), t.TempDir()
	c, _ := New(cfg("https://example.org"))
	dir := c.MusicDir(root)
	audio := bytes.Repeat([]byte{0x42}, 2048)
	old := Song{ID: "original", Artist: "A/B", Title: "Test:Title"}
	create := func(s Song, pin bool) string {
		t.Helper()
		path := FilePath(dir, s.ID)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, audio, 0600); err != nil {
			t.Fatal(err)
		}
		if pin {
			b, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			if err = writeAtomic(pinPath(path), b); err != nil {
				t.Fatal(err)
			}
		}
		return path
	}
	private := create(old, true)
	temp := create(Song{ID: "listened-only"}, false)
	// Leave the first candidate in place as a file belonging to another player.
	personal, err := reserveMP3(shared, private, old)
	if err != nil {
		t.Fatal(err)
	}
	personal = filepath.Join(shared, personal)
	if err = os.WriteFile(personal, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := MigrateDownloads(root, shared); err != nil {
			t.Fatal(err)
		}
	}
	path := PlayablePath(shared, dir, old.ID)
	if filepath.Dir(path) != shared || !strings.HasSuffix(path, " (2).mp3") || !strings.Contains(filepath.Base(path), "A-B - Test-Title") {
		t.Fatal(path)
	}
	if b, err := os.ReadFile(path); err != nil || !bytes.Equal(b, audio) {
		t.Fatal("saved bytes changed", err)
	}
	if _, err := os.Stat(private); !os.IsNotExist(err) {
		t.Fatal("old audio still present", err)
	}
	if b, err := os.ReadFile(personal); err != nil || string(b) != "keep me" {
		t.Fatal("unrelated MP3 changed")
	}
	if !hasAudio(temp) {
		t.Fatal("ordinary listening migrated")
	}
	again, err := c.CacheForPlayback(context.Background(), root, shared, old, 2048, "")
	if err != nil || again != path {
		t.Fatal("playback did not reuse shared audio", again, err)
	}
	if err = c.SaveSong(context.Background(), root, shared, old); err != nil {
		t.Fatal(err)
	}
	if _, err = TrimCache(root, 0, ""); err != nil {
		t.Fatal(err)
	}
	saved, err := DownloadedSongs(root, shared)
	if err != nil || len(saved) != 1 || saved[0].Path != path {
		t.Fatal(saved, err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	other := old
	other.ID = "different"
	create(other, false)
	if err = c.SaveSong(context.Background(), root, shared, other); err != nil {
		t.Fatal(err)
	}
	if PlayablePath(shared, dir, old.ID) != private {
		t.Fatal("a different song claimed the deleted track's name")
	}
	create(old, false)
	if !TemporaryTrack(root, private) {
		t.Fatal("replaying a deleted download must use the limited cache")
	}
	if err = MigrateDownloads(root, shared); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("deleted download was recreated by migration")
	}
}

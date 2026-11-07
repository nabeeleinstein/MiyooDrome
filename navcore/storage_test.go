package navcore

import (
	"bytes"
	"context"
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
	root := t.TempDir()
	a := Album{ID: "album", Name: "Album"}
	songs := []Song{{ID: "a", Title: "First"}, {ID: "b", Title: "Second"}}
	ctx := context.Background()
	if _, err = c.CacheForPlayback(ctx, root, songs[0], DefaultCacheLimit, ""); err != nil {
		t.Fatal(err)
	}
	if err = c.DownloadAlbum(ctx, root, a, songs, nil); err == nil {
		t.Fatal("partial album was accepted")
	}
	albums, _ := DownloadedAlbums(root)
	if len(albums) != 0 {
		t.Fatal("partial album listed")
	}
	saved, _ := DownloadedSongs(root)
	if len(saved) != 1 {
		t.Fatal("completed song missing")
	}
	if _, err = TrimCache(root, 0, ""); err != nil {
		t.Fatal(err)
	}
	bad = false
	if err = c.DownloadAlbum(ctx, root, a, songs, nil); err != nil {
		t.Fatal(err)
	}
	saved, _ = DownloadedSongs(root)
	albums, _ = DownloadedAlbums(root)
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
	root := t.TempDir()
	for _, id := range []string{"a", "b", "c", "d"} {
		if _, err := c.CacheForPlayback(context.Background(), root, Song{ID: id}, 7000, ""); err != nil {
			t.Fatal(err)
		}
		used, err := CacheUsage(root)
		if err != nil || used > 7000 {
			t.Fatal("budget exceeded", used, err)
		}
	}
}

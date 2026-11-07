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
)

func cfg(addr string) Config {
	return Config{Server: addr, Username: "test-user", Salt: "abcd", Token: strings.Repeat("a", 32), MaxBitrate: 192}
}
func TestConfig(t *testing.T) {
	cases := []string{"file:///tmp/test", "https://user:pass@host", "https://host/?q=1", "http://host/#fragment", ""}
	for _, v := range cases {
		c := cfg(v)
		if c.Validate() == nil {
			t.Errorf("accepted bad url %q", v)
		}
	}
	if err := cfg("http://127.0.0.1:4533").Validate(); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cfg("http://localhost"))
	file := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadConfig(file)
	if err != nil {
		t.Fatal(err)
	}
}
func TestEndpointDoesNotAllowParametersToOverrideAuth(t *testing.T) {
	c, e := New(cfg("http://localhost:4533/navidrome/"))
	if e != nil {
		t.Fatal(e)
	}
	u, e := c.Endpoint("stream", map[string][]string{"u": {"attacker"}, "id": {"a&b"}})
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(u, "/navidrome/rest/stream.view") || !strings.Contains(u, "u=test-user") || !strings.Contains(u, "id=a%26b") || strings.Contains(u, "attacker") {
		t.Fatal(u)
	}
}
func TestBrowseAndErrors(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") != strings.Repeat("a", 32) {
			t.Error("auth not sent")
		}
		switch r.URL.Path {
		case "/rest/ping.view":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok"}}`)
		case "/rest/getAlbumList2.view":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","albumList2":{"album":[{"id":"alb1","name":"Alpha","artist":"Anna"}]}}}`)
		case "/rest/getArtists.view":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","artists":{"index":[{"artist":[{"id":"ar1","name":"Anna"}]}]}}}`)
		case "/rest/getArtist.view":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","artist":{"album":[{"id":"alb1","name":"Alpha"}]}}}`)
		case "/rest/getAlbum.view":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","album":{"song":[{"id":"song1","title":"Track One","duration":12}]}}}`)
		case "/rest/getPlaylists.view":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","playlists":{"playlist":[{"id":"p1","name":"Favs"}]}}}`)
		case "/rest/getPlaylist.view":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","playlist":{"entry":[{"id":"song1","title":"Track One"}]}}}`)
		default:
			w.WriteHeader(500)
		}
	}))
	defer s.Close()
	c, _ := New(cfg(s.URL))
	ctx := context.Background()
	if err := c.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	albums, err := c.Albums(ctx, 0)
	if err != nil || len(albums) != 1 || albums[0].Name != "Alpha" {
		t.Fatal(albums, err)
	}
	artists, err := c.Artists(ctx)
	if err != nil || len(artists) != 1 {
		t.Fatal(artists, err)
	}
	aral, err := c.ArtistAlbums(ctx, "ar1")
	if err != nil || len(aral) != 1 {
		t.Fatal(aral, err)
	}
	songs, err := c.Songs(ctx, "alb1")
	if err != nil || len(songs) != 1 {
		t.Fatal(songs, err)
	}
	pls, err := c.Playlists(ctx)
	if err != nil || len(pls) != 1 {
		t.Fatal(pls, err)
	}
	plsongs, err := c.PlaylistSongs(ctx, "p1")
	if err != nil || len(plsongs) != 1 {
		t.Fatal(plsongs, err)
	}
	if _, err := c.Albums(ctx, -1); err == nil {
		t.Fatal("negative offset accepted")
	}
}
func TestStreamCacheID3AndOfflineReuse(t *testing.T) {
	hits := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Query().Get("format") != "mp3" || r.URL.Query().Get("maxBitRate") != "192" {
			t.Fatal("wrong streaming params")
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(append([]byte{0xff, 0xfb, 0x90, 0x64}, bytes.Repeat([]byte{0x55}, 2048)...))
	}))
	defer s.Close()
	c, _ := New(cfg(s.URL))
	dir := filepath.Join(t.TempDir(), "music")
	song := Song{ID: "../../../unsafe", Title: "こんにちは", Artist: "test artist", Album: "test album", Track: 1}
	path, err := c.CacheSong(context.Background(), song, dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("path escaped cache: %s", path)
	}
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if string(data[:3]) != "ID3" || !bytes.Contains(data, []byte("こんにちは")) {
		t.Fatal("tag missing")
	}
	_, err = c.CacheSong(context.Background(), song, dir)
	if err != nil || hits != 1 {
		t.Fatalf("cache miss hits=%d err=%v", hits, err)
	}
}
func TestExistingID3IsNotDuplicated(t *testing.T) {
	orig := append(MakeID3Tag(Song{Title: "original"}), bytes.Repeat([]byte{0x99}, 2048)...)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(orig)
	}))
	defer s.Close()
	c, _ := New(cfg(s.URL))
	path, e := c.CacheSong(context.Background(), Song{ID: "x"}, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(path)
	if !bytes.Equal(data, orig) {
		t.Fatal("double tagged MP3")
	}
}
func TestBadStreamLeavesNoCache(t *testing.T) {
	for _, tc := range []struct {
		ct, body string
		code     int
	}{
		{"application/json", `{"error":"no"}`, 200},
		{"audio/mpeg", "not audio", 200},
		{"audio/mpeg", "error", 500},
	} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", tc.ct)
			w.WriteHeader(tc.code)
			fmt.Fprint(w, tc.body)
		}))
		c, _ := New(cfg(s.URL))
		dir := t.TempDir()
		if _, e := c.CacheSong(context.Background(), Song{ID: "a"}, dir); e == nil {
			t.Fatal("expected failure")
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) != 0 {
			t.Fatal("partial file leftover")
		}
		s.Close()
	}
}
func TestSubsonicErrors(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"subsonic-response":{"status":"failed","error":{"code":40,"message":"Wrong username or password"}}}`)
	}))
	defer s.Close()
	c, _ := New(cfg(s.URL))
	err := c.Ping(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Wrong username") {
		t.Fatal(err)
	}
}

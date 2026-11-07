// Package navcore provides a small Subsonic/OpenSubsonic client for a
// MiyooPod integration. It contains no graphics or audio code.
package navcore

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const ClientName = "MiyooDrome"
const MaxTrackBytes int64 = 64 << 20 // Guard the small SD card against runaway downloads.

type Config struct {
	Server     string `json:"server"`
	Username   string `json:"username"`
	Salt       string `json:"salt"`
	Token      string `json:"token"`
	MaxBitrate int    `json:"max_bitrate"`
}

func (c Config) Validate() error {
	u, e := url.Parse(c.Server)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("Navidrome server must be an http(s) URL without credentials or query")
	}
	if strings.TrimSpace(c.Username) == "" || c.Salt == "" || len(c.Token) != 32 {
		return errors.New("Navidrome auth missing; configure token and salt")
	}
	if _, e := hex.DecodeString(c.Token); e != nil {
		return errors.New("Navidrome token invalid")
	}
	if c.MaxBitrate != 0 && (c.MaxBitrate < 32 || c.MaxBitrate > 320) {
		return errors.New("max_bitrate must be 32-320 kbps")
	}
	return nil
}
func ReadConfig(path string) (Config, error) {
	var c Config
	b, e := os.ReadFile(path)
	if e != nil {
		return c, e
	}
	if e = json.Unmarshal(b, &c); e != nil {
		return c, e
	}
	return c, c.Validate()
}

type Song struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
	Track    int    `json:"track"`
	CoverArt string `json:"coverArt"`
}
type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Album struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
}
type Playlist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type response struct {
	Result struct {
		Status string `json:"status"`
		Error  struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Artists struct {
			Index []struct {
				Artist []Artist `json:"artist"`
			} `json:"index"`
		} `json:"artists"`
		Albums struct {
			Album []Album `json:"album"`
		} `json:"albumList2"`
		Artist struct {
			Album []Album `json:"album"`
		} `json:"artist"`
		Album struct {
			Song []Song `json:"song"`
		} `json:"album"`
		Playlists struct {
			Playlist []Playlist `json:"playlist"`
		} `json:"playlists"`
		Playlist struct {
			Entry []Song `json:"entry"`
		} `json:"playlist"`
	} `json:"subsonic-response"`
}
type Client struct {
	Config Config
	HTTP   *http.Client
}

func New(c Config) (*Client, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 20 * time.Second}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 12 * time.Second,
		MaxIdleConns: 2, IdleConnTimeout: 15 * time.Second,
	}
	return &Client{Config: c, HTTP: &http.Client{Transport: transport, Timeout: 0}}, nil
}
func (c *Client) Endpoint(method string, v url.Values) (string, error) {
	if v == nil {
		v = url.Values{}
	} else {
		n := url.Values{}
		for k, parts := range v {
			n[k] = append([]string(nil), parts...)
		}
		v = n
	}
	for k, val := range map[string]string{"u": c.Config.Username, "s": c.Config.Salt, "t": c.Config.Token, "v": "1.16.1", "c": ClientName, "f": "json"} {
		v.Set(k, val)
	}
	u, e := url.Parse(c.Config.Server)
	if e != nil {
		return "", e
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/rest/" + method + ".view"
	u.RawQuery = v.Encode()
	return u.String(), nil
}
func (c *Client) fetch(ctx context.Context, method string, v url.Values) (*response, error) {
	addr, e := c.Endpoint(method, v)
	if e != nil {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if e != nil {
		return nil, e
	}
	r, e := c.HTTP.Do(req)
	if e != nil {
		return nil, errors.New("Navidrome not reachable; check Wi-Fi, address and port")
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Navidrome HTTP %d", r.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if e != nil {
		return nil, e
	}
	var reply response
	if e = json.Unmarshal(b, &reply); e != nil {
		return nil, errors.New("invalid Navidrome JSON response")
	}
	if reply.Result.Status != "ok" {
		return nil, fmt.Errorf("Navidrome error %d: %s", reply.Result.Error.Code, reply.Result.Error.Message)
	}
	return &reply, nil
}
func (c *Client) Ping(ctx context.Context) error { _, e := c.fetch(ctx, "ping", nil); return e }
func (c *Client) Albums(ctx context.Context, offset int) ([]Album, error) {
	if offset < 0 {
		return nil, errors.New("invalid offset")
	}
	r, e := c.fetch(ctx, "getAlbumList2", url.Values{"type": {"alphabeticalByName"}, "size": {"100"}, "offset": {strconv.Itoa(offset)}})
	if e != nil {
		return nil, e
	}
	return r.Result.Albums.Album, nil
}
func (c *Client) Artists(ctx context.Context) ([]Artist, error) {
	r, e := c.fetch(ctx, "getArtists", nil)
	if e != nil {
		return nil, e
	}
	var all []Artist
	for _, index := range r.Result.Artists.Index {
		all = append(all, index.Artist...)
	}
	return all, nil
}
func (c *Client) ArtistAlbums(ctx context.Context, id string) ([]Album, error) {
	r, e := c.fetch(ctx, "getArtist", url.Values{"id": {id}})
	if e != nil {
		return nil, e
	}
	return r.Result.Artist.Album, nil
}
func (c *Client) Songs(ctx context.Context, id string) ([]Song, error) {
	r, e := c.fetch(ctx, "getAlbum", url.Values{"id": {id}})
	if e != nil {
		return nil, e
	}
	return r.Result.Album.Song, nil
}
func (c *Client) Playlists(ctx context.Context) ([]Playlist, error) {
	r, e := c.fetch(ctx, "getPlaylists", nil)
	if e != nil {
		return nil, e
	}
	return r.Result.Playlists.Playlist, nil
}
func (c *Client) PlaylistSongs(ctx context.Context, id string) ([]Song, error) {
	r, e := c.fetch(ctx, "getPlaylist", url.Values{"id": {id}})
	if e != nil {
		return nil, e
	}
	return r.Result.Playlist.Entry, nil
}

func FilePath(dir, id string) string {
	digest := md5.Sum([]byte(id))
	return filepath.Join(dir, hex.EncodeToString(digest[:])+".mp3")
}
func (c *Client) CacheSong(ctx context.Context, s Song, dir string) (string, error) {
	if s.ID == "" {
		return "", errors.New("song ID empty")
	}
	dest := FilePath(dir, s.ID)
	if info, e := os.Stat(dest); e == nil && info.Size() > 1024 && info.Size() <= MaxTrackBytes {
		return dest, nil
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	rate := c.Config.MaxBitrate
	if rate == 0 {
		rate = 192
	}
	addr, e := c.Endpoint("stream", url.Values{"id": {s.ID}, "format": {"mp3"}, "maxBitRate": {strconv.Itoa(rate)}})
	if e != nil {
		return "", e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if e != nil {
		return "", e
	}
	response, e := c.HTTP.Do(req)
	if e != nil {
		return "", errors.New("Navidrome download failed; check Wi-Fi, address and port")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stream HTTP %d", response.StatusCode)
	}
	if response.ContentLength > MaxTrackBytes {
		return "", errors.New("MP3 exceeds 64 MiB download cap")
	}
	// Never trust a JSON error to be music; some Subsonic servers send HTTP 200 on errors.
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if strings.Contains(contentType, "json") || strings.Contains(contentType, "xml") || strings.Contains(contentType, "text/") {
		return "", fmt.Errorf("stream returned %s, not audio", contentType)
	}
	f, e := os.CreateTemp(dir, ".navidrome-*.part")
	if e != nil {
		return "", e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	// Read a small prefix to check for MP3 sync or ID3 (not just successful HTTP status).
	header := make([]byte, 3)
	if _, e = io.ReadFull(response.Body, header); e != nil {
		f.Close()
		return "", errors.New("empty or truncated MP3 response")
	}
	mp3 := string(header) == "ID3" || (header[0] == 0xff && header[1]&0xe0 == 0xe0)
	if !mp3 {
		f.Close()
		return "", errors.New("stream response is not an MP3")
	}
	var written int64
	if string(header) != "ID3" {
		// Navidrome transcodes often omit tags. Write an ID3v2.4 tag so
		// MiyooPod's on-device library scanner can recover the song metadata.
		tag := MakeID3Tag(s)
		n, err := f.Write(tag)
		if err != nil {
			f.Close()
			return "", err
		}
		written += int64(n)
	}
	n, e := f.Write(header)
	if e != nil {
		f.Close()
		return "", e
	}
	written += int64(n)
	copied, e := io.Copy(f, io.LimitReader(response.Body, MaxTrackBytes-written+1))
	written += copied
	if e != nil {
		f.Close()
		return "", e
	}
	if written > MaxTrackBytes {
		f.Close()
		return "", errors.New("MP3 exceeds 64 MiB download cap")
	}
	if written < 1024 {
		f.Close()
		return "", errors.New("audio response suspiciously short")
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return "", e
	}
	if e = f.Close(); e != nil {
		return "", e
	}
	if e = os.Rename(tmp, dest); e != nil {
		return "", e
	}
	return dest, nil
}

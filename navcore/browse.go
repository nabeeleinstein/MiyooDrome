package navcore

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const SearchPageSize = 40

type SearchResults struct {
	Artists []Artist `json:"artist"`
	Albums  []Album  `json:"album"`
	Songs   []Song   `json:"song"`
}

func (c *Client) Search(ctx context.Context, query string, offset int) (SearchResults, error) {
	params := url.Values{"query": {strings.TrimSpace(query)}}
	for _, kind := range []string{"artist", "album", "song"} {
		params.Set(kind+"Count", strconv.Itoa(SearchPageSize))
		params.Set(kind+"Offset", strconv.Itoa(offset))
	}
	r, err := c.fetch(ctx, "search3", params)
	if err != nil {
		return SearchResults{}, err
	}
	return r.Result.Search, nil
}

func (c *Client) AlbumSongs(ctx context.Context, album Album) ([]Song, error) {
	songs, err := c.Songs(ctx, album.ID)
	for i := range songs {
		if songs[i].CoverArt == "" {
			songs[i].CoverArt = album.CoverArt
		}
		if songs[i].AlbumID == "" {
			songs[i].AlbumID = album.ID
		}
	}
	return songs, err
}

// Filter compilation tracks using all song-level artist credits, and keep album order.
func (c *Client) ArtistSongs(ctx context.Context, artist Artist, albums []Album) ([]Song, error) {
	var all []Song
	seen := make(map[string]bool)
	for _, album := range albums {
		albumCtx, cancel := context.WithTimeout(ctx, 14*time.Second)
		songs, err := c.AlbumSongs(albumCtx, album)
		cancel()
		if err != nil {
			return nil, err
		}
		for _, s := range songs {
			match := s.ArtistID == artist.ID || strings.EqualFold(s.Artist, artist.Name)
			for _, a := range s.Artists {
				if a.ID == artist.ID {
					match = true
					break
				}
			}
			if s.Artist == "" && s.ArtistID == "" && len(s.Artists) == 0 {
				match = strings.EqualFold(album.Artist, artist.Name)
			}
			if match && s.ID != "" && !seen[s.ID] {
				seen[s.ID] = true
				all = append(all, s)
			}
		}
	}
	return all, nil
}

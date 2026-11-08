package main

import (
	"context"
	"image"
	"path/filepath"
	"time"

	"github.com/nabeeleinstein/MiyooDrome/navcore"
	xdraw "golang.org/x/image/draw"
)

type navPreview struct {
	dir, id, title, artist string
	offline                bool
}
type navArtKey struct {
	dir, id string
	offline bool
}
type navArtImage struct {
	key          navArtKey
	large, small *image.RGBA
	retry        time.Time
}
type navArtworkState struct {
	cache   []navArtImage
	pending chan navArtImage
}

func scaledCover(img image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	return dst
}

const navPreviewSize = 160

func (app *MiyooPod) drawNavCover(p navPreview, x, y, size int) {
	state := &app.navArtwork
	key := navArtKey{p.dir, p.id, p.offline}
	for _, cached := range state.cache {
		if cached.key != key {
			continue
		}
		if cached.large != nil {
			img := cached.large
			if size == navPreviewSize {
				img = cached.small
			}
			app.fastBlitImage(img, x, y)
			return
		}
		if time.Now().Before(cached.retry) {
			app.drawNavPlaceholder(x, y, size)
			return
		}
	}
	if state.pending == nil && p.id != "" {
		done := make(chan navArtImage, 1)
		state.pending = done
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			img, err := loadNavCover(ctx, p)
			result := navArtImage{key: key}
			if err == nil {
				result.large = scaledCover(img, COVER_CENTER_SIZE)
				result.small = scaledCover(img, navPreviewSize)
			} else {
				result.retry = time.Now().Add(30 * time.Second)
			}
			done <- result
		}()
	}
	app.drawNavPlaceholder(x, y, size)
}

// Prefer the local image. Both editions use the same server/account directory
// key. A read-only lookup can reuse stable art without sharing credentials.
func loadNavCover(ctx context.Context, p navPreview) (image.Image, error) {
	img, err := navcore.Cover(ctx, nil, p.dir, p.id)
	if err == nil {
		return img, nil
	}
	if MUSIC_ROOT != stableMusicRoot && filepath.Dir(p.dir) == MUSIC_ROOT {
		stableDir := filepath.Join(stableMusicRoot, filepath.Base(p.dir))
		if img, err := navcore.Cover(ctx, nil, stableDir, p.id); err == nil {
			return img, nil
		}
	}
	c, configErr := navClient()
	if configErr != nil || c.MusicDir(MUSIC_ROOT) != p.dir {
		return nil, err
	}
	defer c.HTTP.CloseIdleConnections()
	return navcore.Cover(ctx, c, p.dir, p.id)
}

func (app *MiyooPod) drawNavPlaceholder(x, y, size int) {
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.ProgBG)
	dc.DrawRectangle(float64(x), float64(y), float64(size), float64(size))
	dc.Fill()
	dc.SetFontFace(app.FontSmall)
	dc.SetHexColor(app.CurrentTheme.Dim)
	dc.DrawStringAnchored("Album art", float64(x+size/2), float64(y+size/2), 0.5, 0.5)
}

func (app *MiyooPod) drawNavPreview(p navPreview, x, y, width int) {
	app.drawNavCover(p, x+(width-navPreviewSize)/2, y, navPreviewSize)
	dc := app.DC
	dc.SetFontFace(app.FontSmall)
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.DrawStringAnchored(app.truncateText(p.title, float64(width), app.FontSmall), float64(x+width/2), float64(y+navPreviewSize+26), 0.5, 0.5)
	dc.SetHexColor(app.CurrentTheme.Dim)
	dc.DrawStringAnchored(app.truncateText(p.artist, float64(width), app.FontSmall), float64(x+width/2), float64(y+navPreviewSize+52), 0.5, 0.5)
}

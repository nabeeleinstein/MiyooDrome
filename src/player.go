package main

import (
	"fmt"
	"image"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
)

// playTrackFromList starts playing a track from a list, building a queue from the context
func (app *MiyooPod) playTrackFromList(tracks []*Track, startIdx int) {
	if len(tracks) == 0 || startIdx < 0 || startIdx >= len(tracks) {
		return
	}

	app.Queue.Tracks = make([]*Track, len(tracks))
	copy(app.Queue.Tracks, tracks)
	app.Queue.CurrentIndex = startIdx

	if app.Queue.Shuffle {
		app.buildShuffleOrder(startIdx)
		// buildShuffleOrder puts startIdx at index 0 of the shuffle order
		app.Queue.CurrentIndex = 0
	} else {
		// Clear old shuffle order when playing without shuffle
		app.Queue.ShuffleOrder = nil
	}

	app.playCurrentQueueTrack()

	app.playerReturn = ScreenMenu
	app.setScreen(ScreenNowPlaying)
	app.refreshRootMenu()
	app.drawCurrentScreen()
}

// playCurrentQueueTrack loads and plays the current track in the queue
func (app *MiyooPod) playCurrentQueueTrack() {
	track := app.getCurrentTrack()
	if track == nil {
		return
	}

	if err := app.prepareNavTrack(track); err != nil {
		app.mpvStop()
		app.Playing.State = StateStopped
		app.Playing.Track = track
		app.NPCacheDirty = true
		app.showError(err.Error())
		return
	}

	// A previous track may have ended while its replacement was downloading.
	app.TrackEndPending = false

	app.Playing.Track = track
	app.Playing.State = StatePlaying
	app.Playing.Position = 0
	// Use track duration if available, otherwise will be updated by poller
	if track.Duration > 0 {
		app.Playing.Duration = track.Duration
	} else {
		app.Playing.Duration = 0
	}

	err := app.mpvLoadFile(track.Path)
	if err != nil {
		logMsg(fmt.Sprintf("ERROR: Failed to load: %v", err))
		app.Playing.State = StateStopped
		app.Playing.Track = nil // Clear track on failure
		app.showError(fmt.Sprintf("Failed to load audio\n%s", err.Error()))
		return
	}

	// Verify playback actually started
	time.Sleep(100 * time.Millisecond)
	state := audioGetState()
	if !state.IsPlaying && !state.IsPaused {
		logMsg("ERROR: Audio failed to start playing")
		app.Playing.State = StateStopped
		app.Playing.Track = nil
		app.showError("Playback failed to start")
		return
	}

	app.updateCoverflowForCurrentTrack()
	app.NPCacheDirty = true

	// Reset lyrics state for new track
	app.LyricsCachedTrack = ""
	app.LyricsLastActiveLRC = -1
	app.LyricsScrollOffset = 0
	app.LyricsManualScroll = false

	// Persist queue state when track changes
	app.savePlaybackState()
}

// getCurrentTrack returns the current track based on queue and shuffle state
func (app *MiyooPod) getCurrentTrack() *Track {
	if app.Queue == nil || len(app.Queue.Tracks) == 0 {
		return nil
	}

	idx := app.Queue.CurrentIndex
	if app.Queue.Shuffle && len(app.Queue.ShuffleOrder) > 0 {
		if idx >= 0 && idx < len(app.Queue.ShuffleOrder) {
			idx = app.Queue.ShuffleOrder[idx]
		}
	}

	if idx >= 0 && idx < len(app.Queue.Tracks) {
		return app.Queue.Tracks[idx]
	}
	return nil
}

func (app *MiyooPod) togglePlayPause() {
	if app.Playing == nil {
		return
	}
	if app.Playing.State == StateStopped {
		if app.getCurrentTrack() != nil {
			pos := app.Playing.Position
			app.playCurrentQueueTrack()
			if app.Playing.State == StatePlaying && pos > 0 {
				audioSeek(pos)
				app.Playing.Position = pos
			}
		}
		return
	}
	app.mpvTogglePause()
	// Reflect the mixer state immediately, including in the browsing header.
	if audioIsPaused() {
		app.Playing.State = StatePaused
	} else if audioIsPlaying() {
		app.Playing.State = StatePlaying
	}
	app.MenuBG = nil
	app.NPCacheDirty = true
}

func (app *MiyooPod) nextTrack() {
	if app.Queue == nil || len(app.Queue.Tracks) == 0 {
		return
	}

	// Determine the max index based on shuffle state
	maxIdx := len(app.Queue.Tracks) - 1
	if app.Queue.Shuffle && len(app.Queue.ShuffleOrder) > 0 {
		maxIdx = len(app.Queue.ShuffleOrder) - 1
	}

	app.Queue.CurrentIndex++
	if app.Queue.CurrentIndex > maxIdx {
		if app.Queue.Repeat == RepeatAll {
			app.Queue.CurrentIndex = 0
		} else {
			app.Queue.CurrentIndex = maxIdx
			app.mpvStop()
			app.NPCacheDirty = true
			app.Playing.State = StateStopped
			return
		}
	}

	app.playCurrentQueueTrack()
}

func (app *MiyooPod) prevTrack() {
	if app.Queue == nil || len(app.Queue.Tracks) == 0 {
		return
	}

	if app.Playing != nil && app.Playing.Position > 3.0 {
		app.mpvSeek(-app.Playing.Position)
		app.NPCacheDirty = true // Force UI update when restarting track
		return
	}

	// Determine the max index based on shuffle state
	maxIdx := len(app.Queue.Tracks) - 1
	if app.Queue.Shuffle && len(app.Queue.ShuffleOrder) > 0 {
		maxIdx = len(app.Queue.ShuffleOrder) - 1
	}

	app.Queue.CurrentIndex--
	if app.Queue.CurrentIndex < 0 {
		if app.Queue.Repeat == RepeatAll {
			app.Queue.CurrentIndex = maxIdx
		} else {
			app.Queue.CurrentIndex = 0
		}
	}

	app.playCurrentQueueTrack()
}

func (app *MiyooPod) handleTrackEnd() {
	if app.Queue == nil || len(app.Queue.Tracks) == 0 {
		app.Playing.State = StateStopped
		return
	}

	if app.Queue.Repeat == RepeatOne {
		app.playCurrentQueueTrack()
		return
	}

	// Determine the max index based on shuffle state
	maxIdx := len(app.Queue.Tracks) - 1
	if app.Queue.Shuffle && len(app.Queue.ShuffleOrder) > 0 {
		maxIdx = len(app.Queue.ShuffleOrder) - 1
	}

	app.Queue.CurrentIndex++
	if app.Queue.CurrentIndex > maxIdx {
		if app.Queue.Repeat == RepeatAll {
			app.Queue.CurrentIndex = 0
			app.playCurrentQueueTrack()
		} else {
			app.Queue.CurrentIndex = maxIdx
			app.Playing.State = StateStopped
			app.NPCacheDirty = true
			app.refreshRootMenu()
			app.requestRedraw()
		}
		return
	}

	app.playCurrentQueueTrack()
	app.requestRedraw()
}

func (app *MiyooPod) toggleShuffle() {
	if app.Queue == nil || len(app.Queue.Tracks) == 0 {
		return
	}

	// Remember which track is currently playing
	currentTrack := app.getCurrentTrack()
	var currentPhysicalIdx int = -1
	if currentTrack != nil {
		for i, track := range app.Queue.Tracks {
			if track == currentTrack {
				currentPhysicalIdx = i
				break
			}
		}
	}

	app.Queue.Shuffle = !app.Queue.Shuffle
	if app.Queue.Shuffle {
		// Build shuffle order with current track's physical index
		if currentPhysicalIdx >= 0 {
			app.buildShuffleOrder(currentPhysicalIdx)
		} else {
			app.buildShuffleOrder(0)
		}
		// Reset playback position to start of new shuffle order
		app.Queue.CurrentIndex = 0
		logMsg("INFO: Shuffle enabled")
	} else {
		// When turning off shuffle, set CurrentIndex to physical position
		if currentPhysicalIdx >= 0 {
			app.Queue.CurrentIndex = currentPhysicalIdx
		}
		app.Queue.ShuffleOrder = nil
		logMsg("INFO: Shuffle disabled")
	}
	app.NPCacheDirty = true
}

func (app *MiyooPod) cycleRepeat() {
	if app.Queue == nil {
		return
	}
	switch app.Queue.Repeat {
	case RepeatOff:
		app.Queue.Repeat = RepeatAll
	case RepeatAll:
		app.Queue.Repeat = RepeatOne
	case RepeatOne:
		app.Queue.Repeat = RepeatOff
	}
	app.NPCacheDirty = true
}

func (app *MiyooPod) buildShuffleOrder(startIdx int) {
	n := len(app.Queue.Tracks)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	for i, v := range order {
		if v == startIdx {
			order[0], order[i] = order[i], order[0]
			break
		}
	}
	app.Queue.ShuffleOrder = order
}

func (app *MiyooPod) updateCoverflowForCurrentTrack() {
	if app.Playing == nil || app.Playing.Track == nil {
		return
	}

	track := app.Playing.Track

	if len(app.Coverflow.Albums) == 0 && app.Library != nil {
		app.Coverflow.Albums = app.Library.Albums
	}

	for i, album := range app.Coverflow.Albums {
		if album.Name == track.Album && album.Artist == track.Artist {
			app.Coverflow.CenterIndex = i
			return
		}
		albumArtist := track.AlbumArtist
		if albumArtist == "" {
			albumArtist = track.Artist
		}
		if album.Name == track.Album && album.Artist == albumArtist {
			app.Coverflow.CenterIndex = i
			return
		}
	}
}

// drawNowPlayingScreen renders the Now Playing screen with background caching.
// Static elements are cached as a pixel buffer. Only progress bar redraws each second.
func (app *MiyooPod) drawNowPlayingScreen() {
	dc := app.DC

	if app.Playing == nil || app.Playing.Track == nil {
		dc.SetHexColor(app.CurrentTheme.BG)
		dc.Clear()
		app.drawHeader("Now Playing")
		dc.SetFontFace(app.FontMenu)
		dc.SetHexColor(app.CurrentTheme.Dim)
		dc.DrawStringAnchored("No track playing", SCREEN_WIDTH/2, SCREEN_HEIGHT/2, 0.5, 0.5)
		return
	}

	if app.trackDetails {
		app.drawTrackDetails(app.Playing.Track)
		app.drawStatusBar()
		return
	}
	if app.NowPlayingBG == nil || app.NPCacheDirty {
		app.renderNowPlayingFull()
		app.drawStatusBar()

		if app.NowPlayingBG == nil {
			app.NowPlayingBG = image.NewRGBA(image.Rect(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT))
		}
		copy(app.NowPlayingBG.Pix, app.FB.Pix)
		app.NPCacheDirty = false
	} else {
		copy(app.FB.Pix, app.NowPlayingBG.Pix)
	}

	// Draw progress bar using direct pixel operations (bypass gg)
	app.fastDrawProgressBar(40, PROGRESS_BAR_Y, SCREEN_WIDTH-80, app.Playing.Position, app.Playing.Duration)
}

// updateProgressBarOnly is the fast path called by the playback poller.
// Only updates progress bar region, bypasses gg entirely.
func (app *MiyooPod) updateProgressBarOnly() {
	if app.Playing == nil || app.NowPlayingBG == nil {
		return
	}

	// Don't draw progress bar over lock, volume/brightness, or power-exit overlay
	if app.Locked || app.OverlayVisible || app.PowerExitWarning {
		return
	}

	// Double-check we're still on Now Playing screen
	if app.CurrentScreen != ScreenNowPlaying || app.trackDetails {
		return
	}

	// Restore progress bar region from cache
	fastCopyRegion(app.FB, app.NowPlayingBG, PROGRESS_REGION_Y0, PROGRESS_REGION_Y1)

	// Draw progress bar with direct pixel ops
	app.fastDrawProgressBar(40, PROGRESS_BAR_Y, SCREEN_WIDTH-80, app.Playing.Position, app.Playing.Duration)

	// Full refresh (partial was buggy on Miyoo Mini SDL2)
	app.triggerRefresh()
}

// renderNowPlayingFull draws all static now-playing elements via gg
func (app *MiyooPod) renderNowPlayingFull() {
	dc := app.DC
	track := app.Playing.Track

	dc.SetHexColor(app.CurrentTheme.BG)
	dc.Clear()

	app.drawHeader("Now Playing")
	app.DrawCoverflow()

	const infoX = 254
	const width = SCREEN_WIDTH - infoX - 24
	// The cover and text share a vertical center at all font sizes.
	lines := app.trackTitleLines(track.Title, width)
	lineHeight := float64(app.FontTitle.Metrics().Height.Ceil() + 3)
	height := float64(len(lines)) * lineHeight
	album := track.Album
	if album == "Unknown Album" {
		album = ""
	}
	if track.Artist != "" {
		height += 38
	}
	if album != "" {
		height += 32
	}
	y := float64(COVER_CENTER_Y+COVER_CENTER_SIZE/2) - height/2 + lineHeight/2
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.SetFontFace(app.FontTitle)
	for _, line := range lines {
		dc.DrawStringAnchored(app.truncateText(line, width, app.FontTitle), infoX, y, 0, .5)
		y += lineHeight
	}
	if track.Artist != "" {
		dc.SetFontFace(app.FontArtist)
		dc.DrawStringAnchored(app.truncateText(track.Artist, width, app.FontArtist), infoX, y+6, 0, .5)
		y += 38
	}
	if album != "" {
		dc.SetFontFace(app.FontAlbum)
		dc.DrawStringAnchored(app.truncateText(album, width, app.FontAlbum), infoX, y+2, 0, .5)
	}

	app.drawStatusIndicators(336)
}

func (app *MiyooPod) trackTitleLines(title string, width float64) []string {
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		title = "Unknown title"
	}
	lines := wrapTextLines(title, width, app.FontTitle, app.measureTextCached)
	if len(lines) > 1 {
		lines = []string{lines[0], strings.Join(lines[1:], " ")}
	}
	return lines
}
func (app *MiyooPod) drawTrackDetails(track *Track) {
	dc := app.DC
	dc.SetHexColor(app.CurrentTheme.BG)
	dc.Clear()
	app.drawHeader("Track details")
	dc.SetFontFace(app.FontTitle)
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	for i, line := range app.trackTitleLines(track.Title, 580) {
		dc.DrawStringAnchored(app.truncateText(line, 580, app.FontTitle), 30, float64(74+34*i), 0, .5)
	}
	app.drawFormatTag(track, 30, 145)
	var tags []string
	if track.DiscNum > 0 {
		tags = append(tags, fmt.Sprintf("Disc %d", track.DiscNum))
	}
	if track.TrackNum > 0 {
		number := fmt.Sprintf("Track %d", track.TrackNum)
		if track.TrackTotal > 0 {
			number += fmt.Sprintf(" of %d", track.TrackTotal)
		}
		tags = append(tags, number)
	}
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	dc.SetFontFace(app.FontSmall)
	dc.DrawStringAnchored(strings.Join(tags, " · "), 30, 177, 0, .5)
	dc.SetFontFace(app.FontHeader)
	dc.DrawStringAnchored("Playback controls", 30, 222, 0, .5)
	app.drawButtonLegend(30, 266, "L2 / A", "Play / Pause")
	app.drawButtonLegend(322, 266, "L / R", "Previous / Next")
	app.drawButtonLegend(30, 306, "Hold L / R", "Seek")
	app.drawButtonLegend(322, 306, "Right", "Queue")
	app.drawButtonLegend(30, 346, "SELECT", "Shuffle")
	app.drawButtonLegend(322, 346, "X", "Repeat")
	if track.NavSong != nil && !track.Offline {
		app.drawButtonLegend(30, 386, "Y", "Save song")
	}
	if track.Lyrics != "" {
		app.drawButtonLegend(322, 386, "START", "Lyrics")
	}
}

// drawFormatTag draws a small pill badge showing the audio format (e.g. "FLAC") and
// average bitrate (e.g. "1234 kbps") to the right of it, matching the button hint style.
func (app *MiyooPod) drawFormatTag(track *Track, x, y float64) {
	dc := app.DC
	dc.SetFontFace(app.FontSmall)

	ext := strings.ToLower(filepath.Ext(track.Path))
	var label string
	switch ext {
	case ".flac":
		label = "FLAC"
	case ".ogg":
		label = "OGG"
	case ".mp3":
		label = "MP3"
	default:
		label = strings.ToUpper(strings.TrimPrefix(ext, "."))
	}

	// Badge — same geometry as drawButtonLegend
	labelW, _ := dc.MeasureString(label)
	badgePad := 6.0
	badgeW := labelW + badgePad*2
	badgeH := 20.0
	badgeY := y - badgeH/2

	dc.SetHexColor(app.CurrentTheme.ProgBG)
	dc.DrawRoundedRectangle(x, badgeY, badgeW, badgeH, 4)
	dc.Fill()

	dc.SetHexColor(app.CurrentTheme.HeaderTxt)
	dc.DrawStringAnchored(label, x+badgeW/2, y-1, 0.5, 0.5)

	// Bitrate and sample rate to the right of the badge
	dc.SetHexColor(app.CurrentTheme.ItemTxt)
	if track.Bitrate > 0 && track.SampleRate > 0 {
		dc.DrawStringAnchored(fmt.Sprintf("%d kbps · %g kHz", track.Bitrate, float64(track.SampleRate)/1000), x+badgeW+6, y, 0, 0.5)
	} else if track.Bitrate > 0 {
		dc.DrawStringAnchored(fmt.Sprintf("%d kbps", track.Bitrate), x+badgeW+6, y, 0, 0.5)
	} else if track.SampleRate > 0 {
		dc.DrawStringAnchored(fmt.Sprintf("%g kHz", float64(track.SampleRate)/1000), x+badgeW+6, y, 0, 0.5)
	}
}

# Install MiyooDrome Test

1. Extract the test ZIP and copy its `App/MiyooDromeTest` folder into the SD card’s `App` folder. Open **Apps → MiyooDrome Test**.
2. For the first install, sign in under **Navidrome → Sign in**. To reuse a working login, you can instead copy just `App/MiyooDrome/data/config.json` into `App/MiyooDromeTest/data/`. Create the test data folder if needed. If your server uses a private CA, copy `ca.pem` alongside it. Keep these files on your own card.
3. To try another build, merge the newer `MiyooDromeTest` folder over the existing test folder and keep its `data/`. Builds from GitHub show their short commit identifier under **Settings → About**.

To use a personal TrueType font, place it at `App/MiyooDromeTest/data/ui_font.ttf`. Keep that file when updating. Remove it to return to the bundled Inter font.

## Sign in on the handheld

Search and sign-in use the player’s built-in QWERTY keyboard. Move with the D-pad, press **A** to type, **X** to delete, and **Y** to switch case or symbols. **START** accepts the field or submits the search. **B** goes back and **MENU** cancels.

Under **Navidrome → Sign in**, **L** switches HTTP/HTTPS on the server field. An existing address and username are prefilled. Password entry is masked.

## Browse and resume

- Open **Search** for the full server or use **Search artists** and **Search albums** within those lists. Press **START** to see the results.
- Hold **Down** or **Up** for about three quarters of a second on a long alphabetical list to jump between initials. Tap for a single row.
- Press **L2** while browsing to pause or resume the current song without leaving the list. On **Now Playing**, **A** also toggles playback. **Up** opens track details and controls. **B** returns to the song. Sign-in and search keep their own keyboard controls.
- Press **Y** to queue a song, or highlight an artist in **Downloaded library → Artists** and press **Y** to queue their tracks. **START** opens **Now Playing** while browsing, as in the stable player. On relaunch, press **A** there to resume the saved track and position. Each edition remembers its own listening session. **R2** opens **Now Playing** from an unlocked app screen, including About and download progress. **B** returns.
- Open **Downloaded library** for songs, artists and albums. To remove a complete album saved by the test app, highlight it under **Albums** and press **X** when **Delete album** appears, or open it and choose **Delete downloaded album**. Cancel is selected first. Tracks explicitly saved individually in this build stay available, as do tracks used by another saved album. To keep an individual song from an older album download, press **X** on its online song row before removing the album.
- In **Downloaded library**, highlight a song saved by this app and press **X**. Choose **Delete song from device** to remove its offline copy. **Cancel** is selected first. Other tracks on the album stay available. Download the song or album again to restore it.
- Open **Settings → Storage** to set the playback cache limit or clear old tracks.
- Choose **Settings → Rescan imported music** to index MP3, FLAC, Ogg and M3U/M3U8 playlists throughout `Media/Music/`, including `Media/Music/MiyooDrome/` and nested artist or album folders. Find them under **Downloaded library**. The test app treats the stable player’s files as imported music and uses their saved cover art when available. Rescan once after upgrading to pick up artwork for earlier imports. Navidrome downloads can use artwork embedded in the MP3 or fetched from your server. For an older download with missing art, open the track in Now Playing while connected, then try it offline. The download screen tells you if the audio was saved without offline artwork.

The test player keeps settings, artwork, queues and temporary audio in its own app folder. Explicit downloads go in `Media/Music/MiyooDromeTest/`. To remove the test install, delete `App/MiyooDromeTest/`. You can keep the downloaded MP3s or delete the test music folder separately.

The [player guide](https://github.com/nabeeleinstein/MiyooDrome/blob/test/INSTALL.md) covers controls and playback. Use the test paths above for this edition.

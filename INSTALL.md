# Install MiyooDrome on OnionOS

Use a network-connected handheld such as the Miyoo Mini Plus. The original Miyoo Mini has no built-in Wi-Fi. Previously downloaded tracks can play offline.

1. Extract the install ZIP on a computer. Copy its `App/MiyooDrome` folder into the SD card’s existing `App` folder. For an upgrade, merge the files so your existing `MiyooDrome/data` folder is preserved.
2. Connect the Miyoo Mini Plus to your home Wi-Fi. Open **Apps → MiyooDrome → Navidrome → Sign in**. A working connection from an earlier MiyooDrome installation is picked up automatically.
3. Enter your **server address**, including its port and any reverse-proxy path. Use **L** to switch between HTTPS and HTTP. For example, a home LAN address might be `http://192.168.1.20:4533`. Use the D-pad to highlight a character and **A** to type. **X** deletes, **Y** cycles lowercase, uppercase, and symbols, **B** goes back, and **MENU** cancels.
4. Press **START** to advance to the username, then the password. Press **START** again to check the login and save it. At **Signed in to Navidrome**, press **A** to return. Browse **Albums**, **Artists**, or **Playlists**, and select a song with **A**.

The keyboard supports printable ASCII characters, including spaces. For credentials using other characters, the included desktop helper is an alternative. From the extracted ZIP folder, run `python3 configure.py "/Volumes/YOUR_SD_CARD"` on macOS, or `py -3 configure.py E:\` on Windows, using the card’s actual name or drive. A previous native build’s `App/MiyooPod/data/navidrome.json` can also be copied manually to `App/MiyooDrome/data/config.json`.

The password is masked during entry. The player verifies the login before saving a reusable Subsonic salt and token to the SD card. A failed attempt leaves the previous connection available. Keep the config private. HTTPS protects the credentials on the network. If your server uses a private certificate authority, place the trusted CA certificate in PEM format at `App/MiyooDrome/data/ca.pem`.

## Browse, play, and download

- Open **Search** from the main menu or the Navidrome menu. Type an artist, album, or song using the D-pad keyboard and press **START**. Albums and artists open their menus. Songs start playback.
- Inside an album, **Play album** queues the tracks in disc and track order. Inside an artist, **Play all songs** and **Shuffle artist** queue that artist’s credited tracks from the albums returned by Navidrome.
- **Download album** saves the complete album to the SD card. The progress screen shows the current track. **B** cancels. Start the download again to reuse completed tracks. Finished albums appear in **Downloaded albums** on the main menu and can be played without Wi-Fi or a library rescan.
- Online albums and Now Playing request the artwork supplied by Navidrome. Opening an album or downloading it caches its available cover on the card.
- On first playback, each track is downloaded before it starts. The player requests MP3 at 192 kbps by default, with a 64 MiB limit per track. Navidrome needs a working MP3 transcoder for formats such as FLAC. All downloads, settings and artwork are stored under `App/MiyooDrome/data/`. 

## Manage SD card space

- Highlight a song in an online album, playlist, or search result and press **X** to download it. Press **Y** on Now Playing to keep the current online song. Open **Downloaded songs** for offline playback. Saving a track you already played reuses the existing file.
- Regular listening uses a **256 MiB playback cache**. Under **Storage**, press **A** on **Cache limit** to cycle through 128, 256 and 512 MiB. The player removes the least recently played temporary tracks to make room. Explicit downloads use additional space and stay saved.
- Choose **Storage → Clear old tracks** to remove temporary audio immediately. Saved songs, downloaded albums and the active song are kept. The clear operation also recognizes automatically cached tracks from 0.2.x. To keep an older cached track permanently, select a song in its online album and press **X** before clearing. Manually named music files are left alone.

If the connection check fails, confirm the address and port from another device on the same Wi-Fi. If it reports a login error, open **Sign in** again. If browsing succeeds but a song fails, check SD space and the server’s transcoding log. Local player errors can be found in `App/MiyooDrome/miyoodrome.log`.

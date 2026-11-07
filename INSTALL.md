# Install MiyooDrome on OnionOS

Use a network-connected handheld such as the Miyoo Mini Plus. The original Miyoo Mini has no built-in Wi-Fi. Previously downloaded tracks can play offline.

1. Extract the install ZIP on a computer. Copy its `App/MiyooDrome` folder into the SD card’s existing `App` folder. For an upgrade, merge the files so your existing `MiyooDrome/data` folder is preserved.
2. If you already have a working MiyooDrome `data/config.json`, keep it. A previous native build’s `App/MiyooPod/data/navidrome.json` can also be copied by hand to `App/MiyooDrome/data/config.json`.
3. For a new connection, run the included helper on your computer with Python 3.8 or later. From the extracted ZIP folder on macOS, for example, run `python3 configure.py "/Volumes/YOUR_SD_CARD"`. On Windows, run `py -3 configure.py E:\` using the card’s drive letter. Enter the Navidrome base URL, username, and password when prompted. Use an address reachable from the handheld, such as `http://192.168.1.20:4533` on a trusted home network, or your HTTPS address. Include any reverse-proxy base path.
4. Eject the card and connect OnionOS to the same network. Open **Apps → MiyooDrome → Navidrome → Test connection**. Select **Albums**, **Artists**, or **Playlists**, then select a track with **A**. **B** goes back.

The helper writes a reusable Subsonic salt and token on the card. Keep the config private. HTTPS protects the credentials on the network. If your server uses a private certificate authority, place the trusted CA certificate in PEM format at `App/MiyooDrome/data/ca.pem`.

The first play downloads the full MP3 before starting MiyooPod’s SDL2_mixer engine. The screen can pause during network requests. Each selection plays one track. The default MP3 request is 192 kbps with a 64 MiB limit per file. Navidrome needs a working MP3 transcoder for files it cannot serve directly. For offline playback, use **Navidrome → Rescan offline songs**, then **Songs** in the main menu. Cache, library, settings, and artwork live under `App/MiyooDrome/data/`. To reclaim space, remove unwanted MP3s from its `music` folder and rescan.

If the connection check fails, confirm the address and port from another device on the same Wi-Fi. If it reports a login error, rerun the helper. If browsing succeeds but a song fails, check SD space and the server’s transcoding log. Local player errors can be found in `App/MiyooDrome/miyoodrome.log`.

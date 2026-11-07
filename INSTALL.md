# Install MiyooDrome on OnionOS

Use a network-connected handheld such as the Miyoo Mini Plus. The original Miyoo Mini has no built-in Wi-Fi. Previously downloaded tracks can play offline.

1. Extract the install ZIP on a computer. Copy its `App/MiyooDrome` folder into the SD card’s existing `App` folder. For an upgrade, merge the files so your existing `MiyooDrome/data` folder is preserved.
2. Connect the Miyoo Mini Plus to your home Wi-Fi. Open **Apps → MiyooDrome → Navidrome → Sign in**. A working connection from an earlier MiyooDrome installation is picked up automatically.
3. Enter your **server address**, including its port and any reverse-proxy path. Use **L** to switch between HTTPS and HTTP. For example, a home LAN address might be `http://192.168.1.20:4533`. Use the D-pad to highlight a character and **A** to type. **X** deletes, **Y** cycles lowercase, uppercase, and symbols, **B** goes back, and **MENU** cancels.
4. Press **START** to advance to the username, then the password. Press **START** again to check the login and save it. At **Signed in to Navidrome**, press **A** to return. Browse **Albums**, **Artists**, or **Playlists**, and select a song with **A**.

The keyboard supports printable ASCII characters, including spaces. For credentials using other characters, the included desktop helper is an alternative. From the extracted ZIP folder, run `python3 configure.py "/Volumes/YOUR_SD_CARD"` on macOS, or `py -3 configure.py E:\` on Windows, using the card’s actual name or drive. A previous native build’s `App/MiyooPod/data/navidrome.json` can also be copied manually to `App/MiyooDrome/data/config.json`.

The password is masked during entry. The player verifies the login before saving a reusable Subsonic salt and token to the SD card. A failed attempt leaves the previous connection available. Keep the config private. HTTPS protects the credentials on the network. If your server uses a private certificate authority, place the trusted CA certificate in PEM format at `App/MiyooDrome/data/ca.pem`.

The first play downloads the full MP3 before starting MiyooPod’s SDL2_mixer engine. The screen can pause during network requests. Each selection plays one track. The default MP3 request is 192 kbps with a 64 MiB limit per file. Navidrome needs a working MP3 transcoder for files it cannot serve directly. For offline playback, use **Navidrome → Rescan offline songs**, then **Songs** in the main menu. Cache, library, settings, and artwork live under `App/MiyooDrome/data/`. To reclaim space, remove unwanted MP3s from its `music` folder and rescan.

If the connection check fails, confirm the address and port from another device on the same Wi-Fi. If it reports a login error, open **Sign in** again. If browsing succeeds but a song fails, check SD space and the server’s transcoding log. Local player errors can be found in `App/MiyooDrome/miyoodrome.log`.

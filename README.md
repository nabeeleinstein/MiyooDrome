# MiyooDrome

MiyooDrome plays music from your Navidrome server on a Wi-Fi-connected Miyoo Mini Plus running OnionOS. Songs and albums you download also play offline. MiyooDrome has its own launcher and settings under `App/MiyooDrome/` and can be installed alongside MiyooPod.

## Test builds

Pushes to the `test` branch produce a **MiyooDromeTest-OnionOS** install ZIP. It adds **MiyooDrome Test** alongside the stable player. For a local test build, use `make package-test` with the same toolchain as the stable build. [Test install and update instructions](INSTALL-TEST.md)

## Install and sign in

1. Download the install ZIP from a successful [MiyooDrome build](https://github.com/nabeeleinstein/MiyooDrome/actions/workflows/build-miyoodrome.yml). Unpack the app into the SD card’s `App` folder. When upgrading, merge the folder to keep your existing `data/`.
2. Connect the handheld to Wi-Fi and open **Apps → MiyooDrome → Navidrome → Sign in**. Enter your server address, username and password using the on-screen keyboard. Press **START** to advance and check the connection.
3. Open **Search**, **Albums**, **Artists** or **Playlists**. Press **A** on a song to play it. The player requests MP3 audio from the server, so FLAC libraries work when Navidrome has MP3 transcoding enabled.

The [installation guide](INSTALL.md) covers keyboard controls, HTTPS, offline use and troubleshooting.

In the test app, **Downloaded library** groups songs, albums, artists and imported playlists. New downloads appear immediately. Use **Settings → Rescan imported music** after copying music to the player’s music folder.

## Playback and storage (test branch)

| Action | On the handheld |
| --- | --- |
| Find music | Open **Search** and press **START** after entering an artist, album or song. |
| Play a collection | Choose **Play album**, or open an artist and choose **Play all songs** or **Shuffle artist**. |
| Keep a song | Highlight an online song and press **X**, or press **Y** on Now Playing. |
| Keep an album | Choose **Download album** in its track list. An interrupted download can be resumed by choosing it again. |
| Play offline | Open **Downloaded library → Songs** or **Albums** in the test app. |
| Open the player | Press **R2** from an app screen. **B** returns. |
| Delete a saved album | Open it under **Downloaded library → Albums**, choose **Delete downloaded album** and confirm. |
| Free cache space | Choose **Storage → Clear old tracks**. Use **Cache limit** to switch between 128, 256 (default) and 512 MiB. |

**Saved music** goes into `Media/Music/MiyooDrome/` on the SD card as regular MP3 files. Run a library scan in MiyooPod or another player to pick them up. Version 0.3.1 moves explicit 0.3.0 downloads here automatically on launch. Ordinary playback uses a separate, size-limited cache under `App/MiyooDrome/data/`. Clearing that cache keeps your downloads.

## Credits and license

MiyooDrome is based on **MiyooPod by Danilo Fragoso**, who created the handheld interface, controls and SDL2_mixer playback engine. This adaptation adds Navidrome sign-in, online browsing, search, cover art and downloads. The test app’s About QR opens the MiyooDrome repository. [Original MiyooPod project](https://github.com/danfragoso/miyoopod) · [Upstream documentation](https://github.com/danfragoso/miyoopod#readme)

The original MiyooPod code is copyright © 2026 Danilo Fragoso and is distributed under the MIT License. The install ZIP includes the original copyright notice and the third-party notices. [MIT License](LICENSE) · [Third-party credits and licenses](THIRD_PARTY_LICENSES.md)

## Build

Use Go 1.25.4, Zig 0.13.0 and an ARM-capable strip tool. Packaging uses `llvm-strip` by default. Set `ARM_STRIP` to another compatible tool such as `arm-none-eabi-strip` if needed. Set `CGO_CFLAGS` to include the SDL2 2.28.5 and SDL2_mixer 2.6.3 header directories, then run `make test package`. The ARM build produces `dist/MiyooDrome-OnionOS.zip`. The [build workflow](.github/workflows/build-miyoodrome.yml) lists the exact setup.

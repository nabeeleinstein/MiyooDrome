# MiyooDrome — genuine MiyooPod fork (build kit)

**This repository is source, not a compiled app. The `App/MiyooPod/libs` directory is created from the top-level `libs` directory by GitHub Actions at packaging time.** The GitHub Actions workflow `Build MiyooDrome (ARM / OnionOS)` is configured to compile the native Go+SDL2 program using ARM hard-float GCC. A successful run uploads `MiyooDrome-Native-ARM.zip`; only *that* ZIP is intended for OnionOS installation.

## Build via GitHub Actions

1. Create a new **private** GitHub repository for this project (the source itself contains no credentials).
2. Upload the **contents** of this source ZIP to the repository, preserving `.github/workflows/build-miyoodrome.yml`, `src/`, `navcore/`, `libs/`, and `App/MiyooPod/assets`.
   - GitHub's web uploader may hide dotfolders; verify the workflow file is present. Alternatively on a Mac, unpack ZIP, run `git init`, `git add .`, commit and push to your own repo.
3. In the repository's **Actions** tab, select `Build MiyooDrome (ARM / OnionOS)` and select **Run workflow**. A push to `main` should also trigger it.
4. Open the workflow run. It must show a successful `Compile real MiyooDrome binary` step. Download the `MiyooDrome-Native-ARM` artifact, then extract the `MiyooDrome-Native-ARM.zip` inside it.
5. Back up `/App/MiyooPod` from your Miyoo's SD card. Copy the new `App/MiyooPod` folder onto the card.
6. To reuse credentials from the old working browser, copy `/App/MiyooDrome/data/config.json` to `/App/MiyooPod/data/navidrome.json`. **Do not put credentials into GitHub or upload them into the source ZIP.**
7. Start MiyooPod on OnionOS; the root menu should include `Navidrome`. Choose `Albums`, select a song with **A** and return with **B**. Selected songs are downloaded fully to `/Media/Music/NavidromeCache` before SDL playback. This can take time; the GUI currently blocks during download.

### What is implemented

Native MiyooPod SDL interface, Navidrome menus for artists/albums/playlists, token-based Subsonic API browsing, authenticated MP3 download/cache and playback via MiyooPod's existing SDL audio engine. This is **not** progressive streaming and has not been hardware-tested. Download size guard: 64 MiB per track. Other known limitations: no network-loaded album artwork; Navidrome menu calls can block while loading; playlist playback starts a single selected track, rather than enqueuing full playlist.

### Troubleshooting

- Workflow failure: inspect the first failed step and download its logs; do not install a previous or unmodified binary. Host Go tests do not establish device compatibility.
- No `Navidrome` menu: you are running an old executable; check that the rebuilt binary replaced the previous `/App/MiyooPod/MiyooPod`.
- Missing config: copy the prior `/App/MiyooDrome/data/config.json` to `/App/MiyooPod/data/navidrome.json`.
- Playback failure: try known MP3 track, collect MiyooPod logs and check free SD storage. Album download is required before native playback.

MiyooPod original copyright/license is retained in `LICENSE`; third-party notices are in `THIRD_PARTY_LICENSES.md`.

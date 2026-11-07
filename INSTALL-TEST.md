# Install MiyooDrome Test

1. Extract the test ZIP and copy its `App/MiyooDromeTest` folder into the SD card’s `App` folder. Open **Apps → MiyooDrome Test**.
2. For the first install, sign in under **Navidrome → Sign in**. To reuse a working login, you can instead copy just `App/MiyooDrome/data/config.json` into `App/MiyooDromeTest/data/`. Create the test data folder if needed. If your server uses a private CA, copy `ca.pem` alongside it. Keep these files on your own card.
3. To try another build, merge the newer `MiyooDromeTest` folder over the existing test folder and keep its `data/`. Builds from GitHub show their short commit identifier under **About**.

## Try the sign-in keyboard

Open **Navidrome → Sign in**. On the server and username screens, press **A** to edit with Onion’s keyboard. **START** accepts the text; cancelling the keyboard returns to the field with its previous value. On an already-filled field, **START** skips editing. **L** switches HTTP/HTTPS on the server screen.

The password uses MiyooDrome’s masked QWERTY keyboard. **A** types, **X** deletes, **Y** switches case or symbols, and **START** signs in. If Onion’s keyboard is unavailable, the same built-in layout handles all three fields. You can also select it with **X** before opening Onion’s keyboard.

## Browse and resume

- Open **Search** for the full server or use **Search artists** and **Search albums** within those lists. Confirm the Onion keyboard to see results. If the system keyboard is unavailable, press **START** on the built-in keyboard.
- Hold **Down** or **Up** for about three quarters of a second on a long alphabetical list to jump between initials. Tap for a single row.
- Press **Y** on a song for queue feedback. **R2** opens **Now Playing** from an unlocked app screen, including About and download progress. **B** returns. Finish or cancel Onion’s keyboard before pressing **R2**.
- Open **Downloaded library** for songs, artists and albums. To remove a complete album, open it and choose **Delete downloaded album**. Cancel is selected first. Tracks explicitly saved individually in this build stay available, as do tracks used by another saved album. To keep an individual song from an older album download, save the song with **X** before removing the album.
- After copying your own music or playlists onto the card, choose **Settings → Rescan imported music**. Navidrome downloads and their artwork appear automatically when available.

The test player keeps settings, artwork, queues and temporary audio in its own app folder. Explicit downloads go in `Media/Music/MiyooDromeTest/`. To remove the test install, delete `App/MiyooDromeTest/`. You can keep the downloaded MP3s or delete the test music folder separately.

The [player guide](https://github.com/nabeeleinstein/MiyooDrome/blob/test/INSTALL.md) covers controls and playback. Use the test paths above for this edition.

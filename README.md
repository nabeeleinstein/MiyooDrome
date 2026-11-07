# MiyooDrome

A standalone OnionOS player for a Navidrome library. It uses the native menus, controls, and SDL2_mixer engine from [Danilo Fragoso’s MiyooPod](https://github.com/danfragoso/miyoopod). It installs alongside MiyooPod in `App/MiyooDrome`.

See [installation and home-server setup](INSTALL.md) for the SD card layout and controls. The [upstream README](README_MIYOOPOD.md) describes the original player.

GitHub Actions tests the Subsonic client, builds the 32-bit ARM hard-float player, and packages the app, SDL libraries, setup helper, and licenses. Download the install ZIP from a successful run. For a local build, install Go 1.25.4 and Zig 0.13.0, point `CGO_CFLAGS` at the SDL2 2.28.5 and SDL2_mixer 2.6.3 header directories as in the workflow, then run `make test package`.

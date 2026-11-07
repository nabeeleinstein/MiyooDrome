# MiyooDrome native integration source

Fork based on user-uploaded MiyooPod 0.0.8 source. `src/menu.go` gains a native `Navidrome` entry; `src/navidrome.go` and `navcore/` implement browsing and cached MP3 playback. Original `src/` and SDK libraries are kept.

**Build status**: navcore tests pass on Linux x86_64. The ARM CGO binary cannot be built in this chat sandbox because ARM gcc/sysroot/SDL headers cannot be downloaded (network DNS unavailable). `.github/workflows/build-miyoodrome.yml` provides a build procedure on a normal GitHub Actions Ubuntu runner. It has **not been executed or validated by us**; compilation may uncover further integration issues. No compiled binary is included in this archive, to prevent confusion with the upstream binary. See `INSTALL.md`.

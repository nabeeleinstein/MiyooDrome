#!/bin/sh
# Stage only distributable files, even when packaging from a configured checkout.
set -eu
cd "$(dirname "$0")/.."
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
source_dir=App/MiyooDrome
case "${1:-stable}" in
    stable)
        folder=MiyooDrome
        binary="$source_dir/MiyooDrome"
        cp INSTALL.md LICENSE THIRD_PARTY_LICENSES.md configure.py "$stage/"
        ;;
    test)
        folder=MiyooDromeTest
        binary=dist/test-bin/MiyooDrome
        mkdir -p "$stage/Test-guide"
        cp INSTALL-TEST.md LICENSE THIRD_PARTY_LICENSES.md "$stage/Test-guide/"
        ;;
    *) echo "Unknown package edition: $1" >&2; exit 1 ;;
esac
dest="$stage/App/$folder"
mkdir -p "$dest"
cp "$binary" "$dest/MiyooDrome"
cp "$source_dir/launch.sh" "$dest/"
if [ "$folder" = MiyooDromeTest ]; then
    sed 's/"label": "MiyooDrome"/"label": "MiyooDrome Test"/' "$source_dir/config.json" > "$dest/config.json"
else
    cp "$source_dir/config.json" "$dest/"
fi
cp -R "$source_dir/assets" libs "$dest/"
# Remove build-time debug data only from the distributable copies.
"${ARM_STRIP:-llvm-strip}" --strip-debug "$dest/libs/libSDL2.so" "$dest/libs/libSDL2_mixer.so" "$dest/libs/libSDL2_z.so"
chmod 755 "$dest/MiyooDrome" "$dest/launch.sh"
archive="$PWD/dist/$folder-OnionOS.zip"
mkdir -p dist
(cd "$stage" && zip -q -r "$stage/package.zip" .)
mv "$stage/package.zip" "$archive"

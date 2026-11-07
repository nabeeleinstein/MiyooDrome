#!/bin/sh
# Stage only distributable files, even when packaging from a configured checkout.
set -eu
cd "$(dirname "$0")/.."
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
dest="$stage/App/MiyooDrome"
mkdir -p "$dest"
cp App/MiyooDrome/MiyooDrome App/MiyooDrome/config.json App/MiyooDrome/launch.sh "$dest/"
cp -R App/MiyooDrome/assets libs "$dest/"
chmod 755 "$dest/MiyooDrome" "$dest/launch.sh"
cp INSTALL.md LICENSE THIRD_PARTY_LICENSES.md configure.py "$stage/"
archive="$PWD/dist/MiyooDrome-OnionOS.zip"
mkdir -p dist
(cd "$stage" && zip -q -r "$stage/package.zip" App INSTALL.md LICENSE THIRD_PARTY_LICENSES.md configure.py)
mv "$stage/package.zip" "$archive"

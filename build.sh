#!/bin/sh
# Builds station-volume-<version>.yap: static arm64 native-volume plus the launcher.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
version=$(sed -n 's/^version = "\(.*\)"/\1/p' "$here/plugin.toml")
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/bin"
(cd "$here" && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$stage/bin/native-volume" .)
cp "$here/plugin.toml" "$here/station-volume.sh" "$here/LICENSE" "$stage/"
chmod 755 "$stage/bin/native-volume" "$stage/station-volume.sh"
tar -C "$stage" -czf "$here/station-volume-$version.yap" plugin.toml station-volume.sh LICENSE bin/native-volume
echo "$here/station-volume-$version.yap"

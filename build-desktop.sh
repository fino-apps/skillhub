#!/bin/sh
set -eu
cd "$(dirname "$0")"

npm --prefix frontend ci
VITE_SKILLHUB_DESKTOP=true VITE_DESKTOP_DIRECTORY_PICKER=true VITE_API_BASE='' \
  npm --prefix frontend run build -- --outDir ../backend/cmd/skillhub-server/desktop-assets

export GOTOOLCHAIN=go1.27.4 CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=13.0
export GOFLAGS=-tags=skillhub_desktop
go -C backend run github.com/egoist/mygo/cmd/mygo@v0.10.0 build -platform darwin/amd64 -skip-dmg

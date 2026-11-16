#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
web="$root/backend/cmd/skillhub-server/web"
out="$root/.fino/release"
npm --prefix "$root/frontend" run build
rm -rf "$web"
mkdir -p "$web" "$out"
cp -R "$root/.fino/frontend/." "$web/"
go -C "$root/backend" build -tags desktop -o "$out/skillhub" ./cmd/skillhub-server

#!/bin/sh
set -eu
cd "$(dirname "$0")"

npm --prefix frontend ci
VITE_SKILLHUB_DESKTOP=true VITE_DESKTOP_DIRECTORY_PICKER=true VITE_API_BASE='' \
  npm --prefix frontend run build -- --outDir ../backend/cmd/skillhub-server/desktop-assets

export GOTOOLCHAIN=go1.27.4 CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=13.0
export GOFLAGS=-tags=skillhub_desktop

# Ventura needs plain ditto and MyGo's hdiutil conversion fallback.
if LC_ALL=C /usr/bin/ditto --noclone 2>&1 | grep -q 'unrecognized option'; then
  tools_dir=$(mktemp -d)
  trap 'rm -rf "$tools_dir"' EXIT
  cat > "$tools_dir/ditto" <<'SH'
#!/bin/sh
if [ "${1:-}" = "--noclone" ]; then shift; fi
exec /usr/bin/ditto "$@"
SH
  cat > "$tools_dir/diskutil" <<'SH'
#!/bin/sh
if [ "$*" = 'image create from -h' ]; then exit 1; fi
exec /usr/sbin/diskutil "$@"
SH
  chmod +x "$tools_dir/ditto" "$tools_dir/diskutil"
  export PATH="$tools_dir:$PATH"
fi

go -C backend run github.com/egoist/mygo/cmd/mygo@v0.10.0 build -platform darwin/amd64

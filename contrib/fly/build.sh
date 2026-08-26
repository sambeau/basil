#!/bin/sh
# Cross-compile Basil for the Machine and drop the binary next to the
# Dockerfile, so that `fly deploy` from this directory has a one-line build.
#
# Two flags matter, and neither is optional:
#
#   CGO_ENABLED=0    No C toolchain in the image, no build stage. SQLite is
#                    modernc.org/sqlite and WebP decodes through wazero, so
#                    nothing in Basil actually wants cgo.
#
#   -tags nodynamic  This is the surprising one. CGO_ENABLED=0 alone does NOT
#                    give a static binary here: github.com/gen2brain/webp
#                    (via server/images) pulls in ebitengine/purego, which
#                    dlopen()s a system libwebp if it can find one, and that
#                    leaves the binary asking for glibc's loader at
#                    /lib/ld-linux-*.so.1. On Alpine, whose loader is musl's,
#                    the result is a baffling "not found" for a binary that is
#                    plainly right there. `nodynamic` drops that path and uses
#                    the embedded WASM decoder only — which is what a container
#                    with no libwebp installed would have fallen back to
#                    anyway. `go test -tags nodynamic ./server/images/...`
#                    passes.
#
# GOARCH must match the Machine. Fly's default is amd64; set GOARCH=arm64 here
# if you asked for an ARM Machine.

set -e

cd "$(dirname "$0")"
REPO=$(cd ../.. && pwd)

: "${GOARCH:=amd64}"
export CGO_ENABLED=0 GOOS=linux GOARCH

echo "building basil for linux/$GOARCH..."
(cd "$REPO" && go build -tags nodynamic -o contrib/fly/basil ./cmd/basil)

# A dynamically linked binary here means the nodynamic tag stopped working —
# see above. It will not run on Alpine.
if command -v file >/dev/null 2>&1; then
	file basil | grep -q "statically linked" || {
		echo "WARNING: basil is not statically linked — it will not run on Alpine." >&2
		echo "WARNING: see the note about -tags nodynamic at the top of this script." >&2
	}
fi

ls -lh basil
echo
echo "now: fly deploy"

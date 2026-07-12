#!/bin/sh
# Basil & Parsley installer
#
#   curl -fsSL https://herbaceous.net/install.sh | sh
#
# Downloads the newest release (including prereleases) of basil + pars
# from GitHub Releases, verifies the checksum, and installs to
# /usr/local/bin (or ~/.local/bin if /usr/local/bin isn't writable).
#
# Options (environment variables):
#   BASIL_INSTALL_DIR   Override the install directory
#   BASIL_VERSION       Install a specific version tag (e.g. v1.0.0-alpha.2)

set -eu

REPO="sambeau/basil"
API="https://api.github.com/repos/${REPO}"

say() { printf '%s\n' "$*"; }
die() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || die "curl is required"
command -v tar >/dev/null 2>&1 || die "tar is required"

# --- Detect platform -------------------------------------------------------

OS=$(uname -s)
case "$OS" in
  Darwin) OS=darwin ;;
  Linux) OS=linux ;;
  MINGW* | MSYS* | CYGWIN*)
    die "On Windows, download the .zip from https://github.com/${REPO}/releases and add it to your PATH." ;;
  *) die "Unsupported OS: $OS" ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  x86_64 | amd64) ARCH=amd64 ;;
  arm64 | aarch64) ARCH=arm64 ;;
  *) die "Unsupported architecture: $ARCH" ;;
esac

# --- Find the newest release ----------------------------------------------
# /releases/latest excludes prereleases, so during the alpha we take the
# first entry from /releases instead.

if [ -n "${BASIL_VERSION:-}" ]; then
  TAG="$BASIL_VERSION"
else
  TAG=$(curl -fsSL "${API}/releases?per_page=1" |
    grep -m1 '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  [ -n "$TAG" ] || die "Could not determine the latest release — see https://github.com/${REPO}/releases"
fi

VERSION=${TAG#v}
ASSET="basil_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

say "Installing Basil & Parsley ${TAG} (${OS}/${ARCH})..."

# --- Download and verify ---------------------------------------------------

TMPDIR_INSTALL=$(mktemp -d)
trap 'rm -rf "$TMPDIR_INSTALL"' EXIT

curl -fsSL -o "${TMPDIR_INSTALL}/${ASSET}" "${BASE_URL}/${ASSET}" ||
  die "Download failed: ${BASE_URL}/${ASSET}"
curl -fsSL -o "${TMPDIR_INSTALL}/checksums.txt" "${BASE_URL}/checksums.txt" ||
  die "Download failed: ${BASE_URL}/checksums.txt"

(
  cd "$TMPDIR_INSTALL"
  EXPECTED=$(grep " ${ASSET}\$" checksums.txt | awk '{print $1}')
  [ -n "$EXPECTED" ] || die "No checksum found for ${ASSET}"
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$ASSET" | awk '{print $1}')
  else
    ACTUAL=$(shasum -a 256 "$ASSET" | awk '{print $1}')
  fi
  [ "$EXPECTED" = "$ACTUAL" ] || die "Checksum mismatch for ${ASSET}"
)

tar -xzf "${TMPDIR_INSTALL}/${ASSET}" -C "$TMPDIR_INSTALL"

# --- Install ----------------------------------------------------------------

if [ -n "${BASIL_INSTALL_DIR:-}" ]; then
  INSTALL_DIR="$BASIL_INSTALL_DIR"
elif [ -w /usr/local/bin ]; then
  INSTALL_DIR=/usr/local/bin
else
  INSTALL_DIR="${HOME}/.local/bin"
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "${TMPDIR_INSTALL}/basil" "${INSTALL_DIR}/basil"
install -m 0755 "${TMPDIR_INSTALL}/pars" "${INSTALL_DIR}/pars"

say ""
say "Installed to ${INSTALL_DIR}:"
say "  basil — the web server"
say "  pars  — the Parsley language CLI & REPL"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    say ""
    say "Note: ${INSTALL_DIR} is not on your PATH. Add it with:"
    say "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

say ""
say "Try it out:"
say "  pars -e '\"Hello, \" ++ \"world!\"'"
say ""
say "Docs: https://herbaceous.net"

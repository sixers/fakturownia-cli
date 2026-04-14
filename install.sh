#!/usr/bin/env bash

set -euo pipefail

REPO="${REPO:-sixers/fakturownia-cli}"
BIN_NAME="fakturownia"
BIN_DIR="${BIN_DIR:-${INSTALL_DIR:-$HOME/.local/bin}}"
VERSION_INPUT="${VERSION:-${INSTALL_VERSION:-latest}}"
TOKEN="${GITHUB_TOKEN:-${GH_TOKEN:-}}"

say() {
  printf '%s\n' "$*" >&2
}

die() {
  say "error: $*"
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

cleanup() {
  if [[ -n "${TMPDIR_INSTALL:-}" && -d "${TMPDIR_INSTALL:-}" ]]; then
    rm -rf "$TMPDIR_INSTALL"
  fi
}

trap cleanup EXIT

curl_args=(
  --fail
  --silent
  --show-error
  --location
  --retry 3
  --connect-timeout 10
)

if [[ -z "$TOKEN" ]] && command -v gh >/dev/null 2>&1; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi

if [[ -n "$TOKEN" ]]; then
  curl_args+=(-H "Authorization: Bearer $TOKEN")
fi

download() {
  local url="$1"
  local output="$2"
  curl "${curl_args[@]}" --output "$output" "$url"
}

download_release_asset() {
  local tag="$1"
  local asset_name="$2"
  local output="$3"
  local url="$4"

  if download "$url" "$output"; then
    return
  fi

  if command -v gh >/dev/null 2>&1; then
    say "Falling back to gh release download for $asset_name"
    gh release download "$tag" --repo "$REPO" --pattern "$asset_name" --dir "$TMPDIR_INSTALL" --clobber >/dev/null
    [[ -f "$output" ]] || die "gh release download did not produce $output"
    return
  fi

  die "failed to download $url"
}

fetch_text() {
  local url="$1"
  curl "${curl_args[@]}" "$url"
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *)
      die "unsupported operating system: $(uname -s); only linux and darwin are supported"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *)
      die "unsupported architecture: $(uname -m); only amd64 and arm64 are supported"
      ;;
  esac
}

resolve_version() {
  if [[ "$VERSION_INPUT" != "latest" && -n "$VERSION_INPUT" ]]; then
    printf '%s\n' "$VERSION_INPUT"
    return
  fi

  local response
  response="$(fetch_text "https://api.github.com/repos/$REPO/releases/latest")" || die "failed to query the latest release"
  local version
  version="$(printf '%s' "$response" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [[ -n "$version" ]] || die "could not determine the latest release tag"
  printf '%s\n' "$version"
}

normalize_tag() {
  local version="$1"
  case "$version" in
    v*) printf '%s\n' "$version" ;;
    *) printf 'v%s\n' "$version" ;;
  esac
}

sha256_file() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$path" | awk '{print $1}'
    return
  fi
  die "neither sha256sum nor shasum is available for checksum verification"
}

verify_checksum() {
  local archive="$1"
  local checksum_file="$2"
  local archive_name
  archive_name="$(basename "$archive")"

  local expected
  expected="$(
    awk -v file="$archive_name" '
      {
        name = $2
        sub(/^\*/, "", name)
        if (name == file) {
          print $1
          exit
        }
      }
    ' "$checksum_file"
  )"

  [[ -n "$expected" ]] || die "could not find checksum for $archive_name"

  local actual
  actual="$(sha256_file "$archive")"
  [[ "$actual" == "$expected" ]] || die "checksum verification failed for $archive_name"
}

print_path_hint() {
  if command -v "$BIN_NAME" >/dev/null 2>&1; then
    local resolved
    resolved="$(command -v "$BIN_NAME")"
    if [[ "$resolved" == "$BIN_DIR/$BIN_NAME" ]]; then
      return
    fi
    say "warning: \`$BIN_NAME\` currently resolves to $resolved"
  fi

  case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *)
      say
      say "Add $BIN_DIR to your PATH if needed:"
      say "  export PATH=\"$BIN_DIR:\$PATH\""
      ;;
  esac
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd mktemp

  local os arch version tag archive_version archive_url checksums_url archive checksums
  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"
  tag="$(normalize_tag "$version")"
  archive_version="${tag#v}"

  archive="${BIN_NAME}_${archive_version}_${os}_${arch}.tar.gz"
  archive_url="https://github.com/$REPO/releases/download/$tag/$archive"
  checksums_url="https://github.com/$REPO/releases/download/$tag/checksums.txt"

  TMPDIR_INSTALL="$(mktemp -d)"
  mkdir -p "$BIN_DIR"

  say "Installing $BIN_NAME $tag for $os/$arch"
  say "Downloading $archive"
  download_release_asset "$tag" "$archive" "$TMPDIR_INSTALL/$archive" "$archive_url"

  say "Downloading checksums.txt"
  download_release_asset "$tag" "checksums.txt" "$TMPDIR_INSTALL/checksums.txt" "$checksums_url"

  say "Verifying checksum"
  verify_checksum "$TMPDIR_INSTALL/$archive" "$TMPDIR_INSTALL/checksums.txt"

  say "Extracting archive"
  tar -xzf "$TMPDIR_INSTALL/$archive" -C "$TMPDIR_INSTALL"
  [[ -f "$TMPDIR_INSTALL/$BIN_NAME" ]] || die "archive did not contain $BIN_NAME"

  say "Installing to $BIN_DIR/$BIN_NAME"
  install -m 0755 "$TMPDIR_INSTALL/$BIN_NAME" "$BIN_DIR/$BIN_NAME"

  print_path_hint

  say
  say "Installed $BIN_NAME to $BIN_DIR/$BIN_NAME"
  printf '%s\n' "$BIN_DIR/$BIN_NAME"
}

main "$@"

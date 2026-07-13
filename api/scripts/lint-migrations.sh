#!/usr/bin/env bash
set -euo pipefail

# Pinned squawk 2.59.0 by digest; bump manually (not Dependabot-tracked).
SQUAWK_IMAGE="ghcr.io/sbdchd/squawk@sha256:2a113f17a69f805b8b217b6a9ec7df52eb539c74b492a4941eba8d8eca9919e8"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
API_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# squawk ships amd64 only; force the platform off x86_64.
platform_flag() { [ "$(uname -m)" = "x86_64" ] || printf -- "--platform=linux/amd64"; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
cp "$SCRIPT_DIR/.squawk.toml" "$WORK/.squawk.toml"

# goose bundles Up and Down in one file; Down drops things, so lint Up only.
files=()
for f in "$API_DIR"/migrations/*.sql; do
  base="$(basename "$f")"
  awk '/-- \+goose Up/{u=1;next} /-- \+goose Down/{u=0} u' "$f" >"$WORK/$base"
  files+=("$base")
done

# squawk's image runs as non-root; make the temp dir readable on bind mounts.
chmod -R a+rX "$WORK"

echo "linting Up sections: ${files[*]}"
# shellcheck disable=SC2046,SC2086
docker run --rm $(platform_flag) -v "$WORK:/w" -w /w "$SQUAWK_IMAGE" \
  -c /w/.squawk.toml "${files[@]}"

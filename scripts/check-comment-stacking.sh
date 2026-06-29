#!/usr/bin/env bash
# Flags 2+ consecutive standalone // comment lines; pass files or scan all tracked.
set -euo pipefail

if [ "$#" -gt 0 ]; then
  files="$*"
else
  files=$(git ls-files '*.go' '*.ts' '*.tsx' '*.js' '*.jsx' '*.mjs' '*.cjs')
fi
[ -n "$files" ] || exit 0

# shellcheck disable=SC2086
awk '
  function exempt(s) {
    return s ~ /lint-ignore-comment/ ||
           s ~ /^\/\/(go:|nolint|export)/ ||
           s ~ /^\/\/ ?(eslint-(disable|enable)|@ts-|prettier-ignore|biome-ignore|istanbul )/
  }
  FNR == 1 { run = 0 }
  {
    t = $0; sub(/^[ \t]+/, "", t)
    if (t ~ /^\/\// && !exempt(t)) {
      run++
      if (run >= 2) {
        printf "%s:%d: stacked comments — condense to one pointed line\n", FILENAME, FNR
        bad = 1
      }
    } else {
      run = 0
    }
  }
  END { if (bad) exit 1 }
' $files

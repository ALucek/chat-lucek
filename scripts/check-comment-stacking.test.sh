#!/usr/bin/env bash
# Self-test for check-comment-stacking.sh: real stacks fail, idiomatic patterns pass.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

# Should PASS: single comment, trailing-then-doc, blank-separated, directive-separated.
cat > "$dir/good.go" <<'EOF'
package p
// single standalone is fine
var a int

const b = 1 // trailing comment

// doc after a blank line is not stacked
func f() {}

// first
//nolint:foo
// second
var c int
EOF

# Should FAIL: two directly-consecutive standalone comment lines.
cat > "$dir/bad.go" <<'EOF'
package p
// stacked line one
// stacked line two
var a int
EOF

if ! bash "$here/check-comment-stacking.sh" "$dir/good.go"; then
  echo "FAIL: good.go should pass"; exit 1
fi
if bash "$here/check-comment-stacking.sh" "$dir/bad.go" >/dev/null; then
  echo "FAIL: bad.go should be flagged"; exit 1
fi
echo "comment-stacking self-test: ok"

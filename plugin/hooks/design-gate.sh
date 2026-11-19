#!/usr/bin/env bash
# PreToolUse hook: refuses an Edit or Write that adds an exported Go type no
# design/*.ofn file names, in a module that has a design/ directory. The check
# itself is "godesign hook design-gate" (internal/designgate). Exit status 2
# refuses the edit, with godesign's message as the reason.
set -u

# Every Edit and Write runs this script, so leave at once unless the payload
# names a Go file; godesign re-checks the extension properly.
input="$(cat)"
case "$input" in
*'.go"'*) ;;
*) exit 0 ;;
esac

here="$(cd "$(dirname "$0")" && pwd)"
printf '%s' "$input" | bash "$here/godesign.sh" hook design-gate

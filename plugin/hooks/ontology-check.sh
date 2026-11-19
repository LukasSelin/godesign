#!/usr/bin/env bash
# PreToolUse and PostToolUse hook: "pre" keeps a .ofn file's content before an
# edit, and "post" runs gowl's linter and EL reasoner over the edited file and
# reports what the edit changed. The check itself is "godesign hook
# ontology-check" (internal/ontologycheck).
set -u

# Every Edit and Write runs this script, so leave at once unless the payload
# names a .ofn file; godesign re-checks the extension properly.
input="$(cat)"
case "$input" in
*'.ofn"'* | *'.OFN"'*) ;;
*) exit 0 ;;
esac

here="$(cd "$(dirname "$0")" && pwd)"
printf '%s' "$input" | bash "$here/godesign.sh" hook ontology-check "${1:-post}"

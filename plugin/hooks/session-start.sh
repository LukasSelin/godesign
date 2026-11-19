#!/usr/bin/env bash
# SessionStart hook: in a module that keeps plans in design/, say once when the
# godesign command the other hooks run is missing, rather than letting every
# check skip without a word. Quiet everywhere else.
set -u

root="${CLAUDE_PROJECT_DIR:-$PWD}"
[ -d "$root/design" ] || exit 0
command -v godesign >/dev/null 2>&1 && exit 0
# godesign's own repository builds it from the working tree instead.
grep -qs '^module github.com/LukasSelin/godesign[[:space:]]*$' "$root/go.mod" && exit 0

printf '%s\n' '{"systemMessage": "godesign plugin: the godesign command is not on PATH, so its checks are skipped. Install it with: go install github.com/LukasSelin/godesign/cmd/godesign@latest"}'

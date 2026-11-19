#!/usr/bin/env bash
# Runs godesign for the plugin's other hooks, with the arguments given.
#
# Anywhere but godesign's own repository, that is the godesign on PATH, put
# there by go install. In godesign's repository it is built from the working
# tree first, so that a change to a check is held to the check as changed.
# Each call builds into a directory of its own, because the Stop hooks run at
# the same moment; the go command's cache keeps that cheap.
#
# When there is no godesign to run, it says so on stderr and exits 0, so the
# check is skipped rather than blocking the tool call or the end of a turn.
set -u

root="${CLAUDE_PROJECT_DIR:-$PWD}"
if grep -qs '^module github.com/LukasSelin/godesign[[:space:]]*$' "$root/go.mod"; then
	dir="$(mktemp -d)" || exit 0
	trap 'rm -rf "$dir"' EXIT
	bin="$dir/godesign"
	case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) bin="$bin.exe" ;; esac
	if ! (cd "$root" && go build -o "$bin" ./cmd/godesign) >&2; then
		echo "godesign: the working tree does not build; skipping this check" >&2
		exit 0
	fi
	"$bin" "$@"
	exit $?
fi

if ! command -v godesign >/dev/null 2>&1; then
	echo "godesign: not on PATH; install it with: go install github.com/LukasSelin/godesign/cmd/godesign@latest" >&2
	exit 0
fi
exec godesign "$@"

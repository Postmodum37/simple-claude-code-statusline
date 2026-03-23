#!/bin/bash
# Resolve symlink to find the real script location (where binaries live)
src="${BASH_SOURCE[0]}"
while [[ -L "$src" ]]; do
  dir="$(cd -P "$(dirname "$src")" && pwd)"
  src="$(readlink "$src")"
  [[ "$src" != /* ]] && src="$dir/$src"
done
dir="$(cd -P "$(dirname "$src")" && pwd)"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
[[ "$arch" == "x86_64" ]] && arch="amd64"
[[ "$arch" == "aarch64" ]] && arch="arm64"
exec "$dir/$os-$arch/statusline" "$@" 2>/dev/null || echo ""

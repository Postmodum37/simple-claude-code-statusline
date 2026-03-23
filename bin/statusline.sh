#!/bin/bash
dir="$(dirname "$0")"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
[[ "$arch" == "x86_64" ]] && arch="amd64"
[[ "$arch" == "aarch64" ]] && arch="arm64"
exec "$dir/$os-$arch/statusline" "$@" 2>/dev/null || echo ""

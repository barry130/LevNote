#!/usr/bin/env bash
# Local product gate for LevNote native (local-development phase).
# Builds/vets/tests only the eleven product roots; handoffs evidence payloads
# are deliberately partial snapshots and are never part of the product tree.
set -uo pipefail
cd /f/AIproject/LevNote/native || exit 2

export GOCACHE=F:/AIproject/LevNote/.localtools/cache
export GOTMPDIR=F:/AIproject/LevNote/.localtools/tmp
export GOENV=off GOWORK=off GOFLAGS=-mod=vendor GOPROXY=off GOSUMDB=off
export GOVCS='*:off' GOTOOLCHAIN=local GOTELEMETRY=off CGO_ENABLED=0

GO=../.localtools/go1.27.3/bin/go.exe
ROOTS="./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/..."

cmd="$1"; shift || true
case "$cmd" in
  version) "$GO" version ;;
  build)   "$GO" build $ROOTS ;;
  vet)     "$GO" vet $ROOTS ;;
  test)    "$GO" test "$@" ;;
  fmt)     "$GO" vet $ROOTS ;;
  *) echo "usage: $0 {version|build|vet|test <pkgs>}" >&2; exit 2 ;;
esac

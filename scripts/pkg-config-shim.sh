#!/bin/sh
# pkg-config wrapper used by `make build`.
#
# Why: webview_go has `#cgo pkg-config: gtk+-3.0 webkit2gtk-4.0` hard-coded.
# Ubuntu 24.04 ships only webkit2gtk-4.1. This shim rewrites a request for
# `webkit2gtk-4.0` to `webkit2gtk-4.1` when only 4.1 is available, so the
# build works on both 22.04 (4.0) and 24.04 (4.1).

set -e

mapped=
for arg in "$@"; do
    case "$arg" in
        webkit2gtk-4.0)
            if ! pkg-config --exists webkit2gtk-4.0 2>/dev/null \
               && pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
                arg=webkit2gtk-4.1
            fi
            ;;
    esac
    mapped="$mapped $arg"
done

# shellcheck disable=SC2086
exec pkg-config $mapped

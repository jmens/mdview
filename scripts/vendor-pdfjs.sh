#!/usr/bin/env bash
# Downloads PDF.js distribution into internal/assets/vendor/pdfjs/.
# Run from repo root: scripts/vendor-pdfjs.sh
set -euo pipefail

VERSION="${PDFJS_VERSION:-4.10.38}"
DEST="internal/assets/vendor/pdfjs"
BASE="https://cdn.jsdelivr.net/npm/pdfjs-dist@${VERSION}/build"

mkdir -p "$DEST"
curl -fL --retry 3 -o "$DEST/pdf.min.mjs"        "$BASE/pdf.min.mjs"
curl -fL --retry 3 -o "$DEST/pdf.worker.min.mjs" "$BASE/pdf.worker.min.mjs"

echo "PDF.js $VERSION vendored to $DEST"

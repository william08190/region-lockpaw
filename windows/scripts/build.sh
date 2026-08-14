#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERSION="${1:-1.0.0}"

if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Version must use MAJOR.MINOR.PATCH format" >&2
  exit 2
fi

FILE_VERSION="${VERSION}.0"
BUILD_DIR="${ROOT_DIR}/build/windows-amd64"
DIST_DIR="${ROOT_DIR}/dist"
APP_EXE="${BUILD_DIR}/RegionLockpaw.exe"
SETUP_EXE="${DIST_DIR}/RegionLockpawSetup-${VERSION}-x64.exe"
CHECKSUM_FILE="${DIST_DIR}/SHA256SUMS.txt"

mkdir -p "${BUILD_DIR}" "${DIST_DIR}"
rm -f "${APP_EXE}" "${SETUP_EXE}" "${CHECKSUM_FILE}"

cd "${ROOT_DIR}"
go test ./internal/...
go vet ./internal/geometry
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/platform ./cmd/region-lockpaw

cd "${ROOT_DIR}/cmd/region-lockpaw"
go run github.com/tc-hib/go-winres@v0.3.3 simply \
    --arch amd64 \
    --out rsrc \
    --manifest gui \
    --product-version "${FILE_VERSION}" \
    --file-version "${FILE_VERSION}" \
    --file-description "Region Lockpaw" \
    --product-name "Region Lockpaw" \
    --copyright "Copyright (c) 2026 Region Lockpaw contributors" \
    --original-filename "RegionLockpaw.exe" \
    --icon "${ROOT_DIR}/assets/RegionLockpaw.png"

cd "${ROOT_DIR}"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build \
    -trimpath \
    -ldflags "-s -w -H=windowsgui -X main.version=${VERSION}" \
    -o "${APP_EXE}" \
    ./cmd/region-lockpaw

makensis \
  -DVERSION="${VERSION}" \
  -DFILE_VERSION="${FILE_VERSION}" \
  -DAPP_EXE="${APP_EXE}" \
  -DOUTPUT_EXE="${SETUP_EXE}" \
  "${ROOT_DIR}/installer/RegionLockpaw.nsi"

echo
echo "Built: ${APP_EXE}"
echo "Built: ${SETUP_EXE}"
(
  cd "${BUILD_DIR}"
  shasum -a 256 "RegionLockpaw.exe"
  cd "${DIST_DIR}"
  shasum -a 256 "RegionLockpawSetup-${VERSION}-x64.exe"
) | tee "${CHECKSUM_FILE}"
echo "Checksums: ${CHECKSUM_FILE}"

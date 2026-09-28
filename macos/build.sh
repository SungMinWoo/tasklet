#!/bin/bash
# SwiftUI 메뉴바 앱 빌드 — Xcode 없이 명령줄 도구만 쓴다.
# 사용법: macos/build.sh [--run]
set -euo pipefail
cd "$(dirname "$0")"
root=$(cd .. && pwd)
app="$root/bin/Tasklet.app"
sdk=$(xcrun --show-sdk-path)

# Go CLI가 먼저 있어야 한다 (앱은 화면만 담당하고 쓰기는 CLI를 부른다).
if [ ! -x "$root/bin/tasklet" ]; then
  echo "bin/tasklet이 없다 → go build -o bin/tasklet ./cmd/tasklet 먼저 실행" >&2
  exit 1
fi

rm -rf "$app"
mkdir -p "$app/Contents/MacOS"

swiftc -sdk "$sdk" -target arm64-apple-macos13.0 -parse-as-library -O \
  -o "$app/Contents/MacOS/Tasklet" Sources/*.swift

cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleExecutable</key><string>Tasklet</string>
  <key>CFBundleIdentifier</key><string>local.tasklet.menubar</string>
  <key>CFBundleName</key><string>tasklet</string>
  <key>CFBundleShortVersionString</key><string>0.1</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSUIElement</key><true/>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
</dict></plist>
PLIST

echo "빌드됨: $app"

if [ "${1:-}" = "--run" ]; then
  pkill -x Tasklet 2>/dev/null || true
  sleep 0.5
  open "$app"
  echo "실행함 — 메뉴바 오른쪽을 확인"
fi

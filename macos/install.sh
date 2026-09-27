#!/bin/bash
# 설치 — 실행 파일을 데스크탑 밖으로 옮기고 로그인 시 자동 실행을 건다.
#
#   macos/install.sh            설치(또는 갱신) + 실행
#   macos/install.sh --uninstall  자동 실행 해제 + 설치한 것 제거
#
# 왜 밖으로 옮기나: 앱이 ~/Desktop 안의 파일을 실행하면 macOS가 매번
# "데스크탑 폴더 접근" 권한을 물어본다 (실측 2026-09-23).
set -euo pipefail
cd "$(dirname "$0")/.."
repo=$(pwd)

bin_dir="$HOME/.local/bin"
app_dir="$HOME/Applications"
app="$app_dir/Tasklet.app"
agent="$HOME/Library/LaunchAgents/local.tasklet.menubar.plist"
label="local.tasklet.menubar"

if [ "${1:-}" = "--uninstall" ]; then
  launchctl bootout "gui/$UID/$label" 2>/dev/null || true
  rm -f "$agent"
  pkill -x Tasklet 2>/dev/null || true
  rm -rf "$app"
  rm -f "$bin_dir/tasklet"
  echo "제거함: 자동 실행, $app, $bin_dir/tasklet"
  echo "데이터(~/.tasklet)는 그대로 둔다."
  exit 0
fi

mkdir -p "$bin_dir" "$app_dir"

echo "1/4 Go CLI 빌드 → $bin_dir/tasklet"
go build -o "$bin_dir/tasklet" ./cmd/tasklet

echo "2/4 메뉴바 앱 빌드 → $app"
./macos/build.sh >/dev/null
rm -rf "$app"
cp -R "$repo/bin/Tasklet.app" "$app"

echo "3/4 로그인 시 자동 실행 등록"
cat > "$agent" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>$label</string>
  <key>ProgramArguments</key>
  <array>
    <string>$app/Contents/MacOS/Tasklet</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>TASKLET_BIN</key><string>$bin_dir/tasklet</string>
  </dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><false/>
  <key>ProcessType</key><string>Interactive</string>
</dict></plist>
PLIST

# bootout은 비동기다. 곧바로 bootstrap하면 아직 안 내려가서
# "Bootstrap failed: 5: Input/output error"가 난다 (실측 2026-09-27, 재설치할 때마다).
# 서비스가 사라지는 것을 확인한 뒤 등록하고, 그래도 실패하면 몇 번 더 해본다.
launchctl bootout "gui/$UID/$label" 2>/dev/null || true
for _ in $(seq 25); do
  launchctl print "gui/$UID/$label" >/dev/null 2>&1 || break
  sleep 0.2
done

registered=no
for attempt in 1 2 3; do
  if launchctl bootstrap "gui/$UID" "$agent" 2>/dev/null; then
    registered=yes
    break
  fi
  [ "$attempt" = 3 ] || sleep 1
done

echo "4/4 실행"
pkill -x Tasklet 2>/dev/null || true
sleep 0.5
if [ "$registered" = yes ]; then
  launchctl kickstart -k "gui/$UID/$label"
else
  # 등록이 안 돼도 앱은 띄운다 (자동 실행만 빠진 상태).
  echo "자동 실행 등록 실패 — 이번에는 앱만 띄운다." >&2
  echo "  다시 걸려면: launchctl bootout gui/$UID/$label; launchctl bootstrap gui/$UID $agent" >&2
  open "$app"
fi

echo
echo "끝. 메뉴바 오른쪽을 확인하세요."
echo "  실행 파일: $bin_dir/tasklet"
echo "  앱:        $app"
echo "  자동 실행: $agent"

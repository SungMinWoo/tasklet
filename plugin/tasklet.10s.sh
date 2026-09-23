#!/bin/bash
# SwiftBar 플러그인 — 파일명의 10s = 갱신 주기 (DESIGN.md 7장)
# 설치: 이 파일을 SwiftBar 플러그인 폴더에 링크한다.
#   ln -s "$PWD/plugin/tasklet.10s.sh" ~/SwiftBar/tasklet.10s.sh
# 링크로 실행되므로 링크를 풀어 저장소 위치를 찾는다.
src="${BASH_SOURCE[0]}"
while [ -L "$src" ]; do
  link="$(readlink "$src")"
  case "$link" in
    /*) src="$link" ;;
    *) src="$(dirname "$src")/$link" ;;
  esac
done
root="$(cd "$(dirname "$src")/.." && pwd)"
exec "$root/bin/tasklet" menu

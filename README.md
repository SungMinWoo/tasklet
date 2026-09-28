# tasklet

macOS 메뉴바에서 쓰는 개인 업무 메모입니다. 한 줄로 적으면 Claude Haiku가 업무·요청자·기한을 뽑아 정리합니다.

[English](README.en.md)

![tasklet 팝오버](docs/popover.png)

## 소개

들은 말을 그대로 적으면 됩니다.

```
"성민우 선임이 화요일까지 회원가입 업무 진행하라고 함"
  →  회원가입 기능 개발 · 성민우 선임 · 9/29(화) D-2
```

메뉴바에는 지난 일과 오늘 할 일의 개수가 늘 보입니다. 누르면 기한이 급한 순서로 목록이 열립니다.

## 동작 방식

화면은 Swift가 그리고, 나머지는 Go CLI가 맡습니다.

```
Tasklet.app (SwiftUI)        메뉴바 · 팝오버. 10초마다 상태를 다시 읽습니다
        │
        ▼  tasklet <명령> --json
tasklet (Go CLI)             파싱 · 날짜 계산 · 저장 · 잠금
        ├──→ claude -p --model haiku      문장 해석 (3~12초)
        └──→ ~/.tasklet/tasks.json        flock + 임시파일 rename
```

날짜 계산은 LLM에 맡기지 않습니다. Haiku는 `{"kind":"weekday","value":"tue"}` 처럼 의미만 뽑고, 실제 날짜는 Go가 계산합니다. LLM이 자주 틀리는 부분이라 테스트로 고정해 두었습니다.

## 필요한 것

- Apple Silicon Mac, macOS 13 이상. Xcode는 필요하지 않고 명령줄 도구만 있으면 됩니다
- Go 1.25 이상
- [Claude Code](https://claude.com/claude-code) CLI. 구독 로그인 상태면 되고 API 키는 따로 필요하지 않습니다

## 설치

```bash
git clone https://github.com/SungMinWoo/tasklet.git
cd tasklet
./macos/install.sh
```

CLI는 `~/.local/bin/tasklet`, 앱은 `~/Applications/Tasklet.app`에 설치되고 로그인할 때 자동으로 실행됩니다.
제거는 `./macos/install.sh --uninstall` 입니다. 데이터는 지우지 않습니다.

## 사용법

메뉴바 캐릭터를 누르면 팝오버가 열립니다.

- 아래 입력창에 한 줄 적고 `⏎`
- 여러 줄을 한 번에 넣으려면 **☰** 를 누르고 붙여넣은 뒤 `⌘⏎`. 줄마다 한 건으로 들어갑니다. 줄마다 동시에 파싱해서 세 줄이든 한 줄이든 걸리는 시간은 비슷합니다
- 행에 마우스를 올리면 **기한 · 고쳐 쓰기 · 삭제** 버튼이 나옵니다. 왼쪽 **○** 을 누르면 완료됩니다
- 기한이 애매하면(예: "화요일"이 오늘인 경우) 확인 카드로 알려줍니다

터미널에서도 같은 일을 할 수 있습니다.

| 명령 | 하는 일 |
|---|---|
| `tasklet add "<자연어>"` | 한 줄 추가 |
| `tasklet add --batch [텍스트]` | 여러 줄 한 번에 추가. 텍스트가 없으면 표준입력에서 읽습니다 |
| `tasklet list [--all]` | 목록. `--all` 은 완료한 것까지 |
| `tasklet done <id>` | 완료 |
| `tasklet delete <id>...` | 삭제. id를 여러 개 줄 수 있습니다 |
| `tasklet edit <id>` | 원래 문장을 고쳐 다시 파싱 |
| `tasklet due <id> <spec>` | 기한만 변경 (`+1d` `eow` `next_eow` `none` 등) |
| `tasklet theme <name>` · `tasklet mascot <name>` | 테마 5종 · 캐릭터 7종 |

## 데이터

| 무엇 | 어디 |
|---|---|
| 할 일 | `~/.tasklet/tasks.json` |
| 설정 (테마·캐릭터) | `~/.tasklet/config.json` |

둘 다 사람이 읽고 고칠 수 있는 JSON입니다. 쓰기는 `flock` 으로 잠근 뒤 임시파일을 `rename` 하는 방식이라, 중간에 끊겨도 반쪽 파일이 남지 않습니다.

## 개발

```bash
go test ./...                 # 113개 케이스 (날짜 계산 · 저장 · 동시성 · 응답 파싱)
go build -o bin/tasklet ./cmd/tasklet
./macos/build.sh --run        # SwiftUI 앱 빌드 후 실행
```

## 라이선스

[MIT](LICENSE)

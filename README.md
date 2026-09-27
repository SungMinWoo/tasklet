# tasklet

macOS 메뉴바에 사는 개인 업무 메모. 대충 한 줄 적으면 Claude Haiku가 업무·요청자·기한을 뽑아 정리한다.

**[한국어](#한국어) · [English](#english)**

---

## 한국어

### 이게 뭔가

회의 중에 들은 말을 그대로 적으면 된다.

```
"성민우 선임이 화요일까지 회원가입 업무 진행하라고 함"
   ↓
회원가입 기능 개발 · 성민우 선임 · 9/29(화) D-2
```

메뉴바에는 지난 일과 오늘 할 일의 개수가 늘 보이고, 누르면 기한이 급한 순서로 정리된 목록이 열린다.

### 어떻게 돌아가나

```
[화면]  Tasklet.app          SwiftUI MenuBarExtra. 그리기만 한다
   │    10초마다 state를 다시 읽는다
   ▼
[처리]  tasklet (Go CLI)     파싱·날짜 계산·저장·잠금
   ├──→ claude -p --model haiku    문장 해석 (3~12초)
   └──→ ~/.tasklet/tasks.json      flock + 임시파일 rename
```

**날짜 계산은 LLM에 맡기지 않는다.** Haiku는 `{"kind":"weekday","value":"tue"}` 처럼 의미만 뽑고,
"그래서 며칠인지"는 Go가 계산한다. LLM이 자주 틀리는 부분이라 테스트로 고정해 뒀다.

### 필요한 것

- Apple Silicon Mac, macOS 13 이상 (Xcode는 필요 없다 — 명령줄 도구만 있으면 된다)
- Go 1.25 이상
- [Claude Code](https://claude.com/claude-code) CLI — 구독 로그인 상태면 되고, API 키는 필요 없다

### 설치

```bash
git clone https://github.com/SungMinWoo/tasklet.git
cd tasklet
./macos/install.sh
```

CLI는 `~/.local/bin/tasklet`, 앱은 `~/Applications/Tasklet.app`에 설치되고 로그인할 때 자동으로 뜬다.
지울 때는 `./macos/install.sh --uninstall` (데이터는 남는다).

### 쓰는 법

메뉴바 캐릭터를 누르면 팝오버가 열린다.

- 맨 아래 입력창에 한 줄 적고 `⏎`
- 여러 줄을 한 번에 넣으려면 **☰** — 붙여넣고 `⌘⏎`. 줄마다 한 건으로 들어가고, 줄마다 동시에 파싱해서 3줄이든 한 줄이든 비슷하게 걸린다
- 행에 마우스를 올리면 **📅 기한 · ✏️ 고쳐 쓰기 · 🗑 삭제**, 왼쪽 **○** 을 누르면 완료
- 기한이 애매하면("화요일"이 오늘이면?) 확인 카드로 알려준다

터미널에서도 된다.

| 명령 | 하는 일 |
|---|---|
| `tasklet add "<자연어>"` | 한 줄 추가 |
| `tasklet add --batch [텍스트]` | 여러 줄 한 번에 (없으면 표준입력에서 읽는다) |
| `tasklet list [--all]` | 목록 (`--all`은 완료한 것까지) |
| `tasklet done <id>` | 완료 |
| `tasklet delete <id>...` | 삭제 (여러 개 가능) |
| `tasklet edit <id>` | 원래 문장을 고쳐 다시 파싱 |
| `tasklet due <id> <spec>` | 기한만 변경 (`+1d` `eow` `next_eow` `none` …) |
| `tasklet theme <name>` / `tasklet mascot <name>` | 테마 5종 · 캐릭터 7종 |

### 데이터

| 무엇 | 어디 |
|---|---|
| 할 일 | `~/.tasklet/tasks.json` |
| 설정 (테마·캐릭터) | `~/.tasklet/config.json` |

둘 다 사람이 읽고 고칠 수 있는 JSON이다. 쓰기는 `flock` + 임시파일 `rename`이라 중간에 끊겨도 반쪽 파일이 남지 않는다.

### 개발

```bash
go test ./...                 # 113개 케이스 (날짜 계산·저장·동시성·응답 파싱)
go build -o bin/tasklet ./cmd/tasklet
./macos/build.sh --run        # SwiftUI 앱 빌드 후 실행
```

설계 결정과 실측 기록은 [DESIGN.md](DESIGN.md)에 있다 — 왜 SQLite가 아닌 JSON인지, 왜 날짜를 Go가 계산하는지, `claude -p` 호출 옵션을 어떻게 골랐는지.

---

## English

### What it is

A personal task memo that lives in the macOS menu bar. Jot down what you heard in a meeting, as you heard it:

```
"Sung asked me to finish the signup work by Tuesday"
   ↓
Signup feature · Sung · Tue 9/29 (D-2)
```

The menu bar always shows how many tasks are overdue and due today. Click it and the list opens, most urgent first.

### How it works

```
[UI]     Tasklet.app         SwiftUI MenuBarExtra. Draws only
   │     re-reads state every 10s
   ▼
[Logic]  tasklet (Go CLI)    parsing, date math, storage, locking
   ├──→ claude -p --model haiku    understands the sentence (3-12s)
   └──→ ~/.tasklet/tasks.json      flock + atomic rename
```

**Date math is not left to the LLM.** Haiku only extracts meaning — `{"kind":"weekday","value":"tue"}` —
and Go turns that into an actual date. That is the part LLMs get wrong, so it is pinned down by tests.

### Requirements

- Apple Silicon Mac, macOS 13+ (no Xcode needed — command line tools are enough)
- Go 1.25+
- [Claude Code](https://claude.com/claude-code) CLI — a signed-in subscription works; no API key needed

### Install

```bash
git clone https://github.com/SungMinWoo/tasklet.git
cd tasklet
./macos/install.sh
```

The CLI goes to `~/.local/bin/tasklet`, the app to `~/Applications/Tasklet.app`, and it starts at login.
To remove it: `./macos/install.sh --uninstall` (your data stays).

### Usage

Click the character in the menu bar to open the popover.

- Type one line in the field at the bottom and hit `⏎`
- For several at once, click **☰**, paste, and hit `⌘⏎`. One line becomes one task, and lines are parsed concurrently — three lines take about as long as one
- Hover a row for **📅 due · ✏️ rewrite · 🗑 delete**; click the **○** on the left to complete it
- When a due date is ambiguous (what if "Tuesday" is today?), a confirmation card says so

It works from the terminal too.

| Command | What it does |
|---|---|
| `tasklet add "<sentence>"` | Add one task |
| `tasklet add --batch [text]` | Add many at once (reads stdin if no text) |
| `tasklet list [--all]` | List (`--all` includes completed) |
| `tasklet done <id>` | Complete |
| `tasklet delete <id>...` | Delete (accepts several ids) |
| `tasklet edit <id>` | Rewrite the original sentence and parse again |
| `tasklet due <id> <spec>` | Change only the due date (`+1d` `eow` `next_eow` `none` …) |
| `tasklet theme <name>` / `tasklet mascot <name>` | 5 themes · 7 characters |

### Data

| What | Where |
|---|---|
| Tasks | `~/.tasklet/tasks.json` |
| Settings (theme, character) | `~/.tasklet/config.json` |

Both are plain JSON you can read and edit. Writes use `flock` plus an atomic `rename`, so an interrupted write never leaves half a file.

### Development

```bash
go test ./...                 # 113 cases (date math, storage, concurrency, response parsing)
go build -o bin/tasklet ./cmd/tasklet
./macos/build.sh --run        # build the SwiftUI app and run it
```

Design decisions and measurements live in [DESIGN.md](DESIGN.md) (Korean) — why JSON instead of SQLite, why Go does the date math, and how the `claude -p` invocation options were chosen.

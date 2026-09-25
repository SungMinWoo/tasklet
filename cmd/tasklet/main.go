// tasklet — 메뉴바 업무 메모 (DESIGN.md 4장)
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SungMinWoo/tasklet/internal/parse"
	"github.com/SungMinWoo/tasklet/internal/store"
)

const usage = `사용법:
  tasklet add "<자연어>"    할 일 추가 (Haiku 파싱, 3~12초)
  tasklet list              남은 일 목록
  tasklet list --all        완료한 일까지
  tasklet done <id>         완료 처리
  tasklet delete <id>       목록에서 지움 (되돌릴 수 없음)
  tasklet menu              SwiftBar 메뉴 출력
  tasklet prompt            입력창을 띄워 추가
  tasklet edit <id>         원래 문장을 고쳐 다시 파싱
  tasklet due <id> <spec>   기한만 변경 (+0d +1d eow next_eow none ...)
  tasklet theme <name>      테마 변경
  tasklet mascot <name>     캐릭터 변경
  tasklet state --json      화면(SwiftUI)이 읽는 상태 JSON
  --json                    add·done·delete·due·edit·theme·mascot에 붙이면 결과를 상태 JSON으로 낸다
`

// jsonMode면 결과를 사람이 읽는 줄 대신 화면(SwiftUI)이 읽는 JSON으로 낸다.
var jsonMode bool

func main() {
	os.Args = takeJSONFlag(os.Args)
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "add":
		err = cmdAdd(os.Args[2:])
	case "list":
		err = cmdList(os.Args[2:])
	case "done":
		err = cmdDone(os.Args[2:])
	case "delete":
		err = cmdDelete(os.Args[2:])
	case "menu":
		err = cmdMenu()
	case "state":
		err = cmdState()
	case "prompt":
		err = cmdPrompt()
	case "edit":
		err = cmdEdit(os.Args[2:])
	case "due":
		err = cmdDue(os.Args[2:])
	case "theme":
		err = cmdSetting("theme", os.Args[2:])
	case "mascot":
		err = cmdSetting("mascot", os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "모르는 명령: %s\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if errors.Is(err, errCancelled) {
		return // 사용자가 취소한 것은 오류가 아니다
	}
	if err != nil {
		if jsonMode {
			jsonError(err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "오류:", err)
		os.Exit(1)
	}
}

// takeJSONFlag는 어디에 있어도 --json을 빼내고 jsonMode를 켠다.
func takeJSONFlag(args []string) []string {
	out := args[:0:0]
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
			continue
		}
		out = append(out, a)
	}
	return out
}

func cmdAdd(args []string) error {
	sentence := strings.TrimSpace(strings.Join(args, " "))
	if sentence == "" {
		return fmt.Errorf(`추가할 문장이 없다. 예: tasklet add "화요일까지 회원가입 개발"`)
	}
	res, resolved, err := interpret(sentence)
	if err != nil {
		return err
	}
	id, err := saveNew(sentence, res, resolved)
	if err != nil {
		return err
	}
	reasons := parse.Reasons(sentence, res, resolved)
	if jsonMode {
		f, rerr := store.Read()
		if rerr != nil {
			return rerr
		}
		return printState(f, reasons, id)
	}
	fmt.Printf("%d번 추가: %s\n", id, describe(taskOf(sentence, res, resolved), time.Now()))
	// 확인창은 prompt·edit에서 띄운다. 터미널에서는 이유만 알린다 (DESIGN.md 8장).
	for _, r := range reasons {
		fmt.Printf("  · %s\n", r)
	}
	return nil
}

// interpret은 문장 하나를 Haiku로 보내 기한까지 계산한다.
// 실패해도 크래시 없이 기본값으로 내려간다 (DESIGN.md 6장 방어 코드).
func interpret(sentence string) (parse.Result, parse.Resolved, error) {
	if err := parse.LookupClaude(); err != nil {
		return parse.Result{}, parse.Resolved{}, err
	}
	today := time.Now()
	res, err := parse.Extract(context.Background(), sentence, today)
	if err != nil {
		// 조용히 실패하지 않는다: 기본값으로 저장하고 알린다 (DESIGN.md 6장 방어 코드 3번).
		fmt.Fprintf(os.Stderr, "파싱 실패 — 기한 없이 저장한다: %v\n", err)
		res = parse.Fallback(sentence)
	}

	resolved, rerr := parse.ResolveDue(res.Due, today)
	if rerr != nil {
		// Haiku가 스키마에 없는 kind를 보내는 일이 있다 (예: "unsure").
		// 기한 없음으로 낮추고 애매한 것으로 표시해, 확인창에서 사용자가 고치게 한다.
		fmt.Fprintf(os.Stderr, "기한을 해석하지 못했다: %v\n", rerr)
		resolved = parse.Resolved{}
		res.Due = parse.Due{Kind: "none"}
		res.Unsure = true
	}

	return res, resolved, nil
}

func taskOf(sentence string, res parse.Result, resolved parse.Resolved) store.Task {
	return store.Task{
		Title:     res.Title,
		Requester: strPtr(res.From),
		DueAt:     datePtr(resolved.Date),
		Raw:       sentence,
		CreatedAt: time.Now(),
	}
}

func saveNew(sentence string, res parse.Result, resolved parse.Resolved) (int, error) {
	var id int
	err := store.Update(func(f *store.File) error {
		id = f.Add(taskOf(sentence, res, resolved))
		return nil
	})
	return id, err
}

func cmdList(args []string) error {
	all := len(args) > 0 && args[0] == "--all"
	f, err := store.Read()
	if err != nil {
		return err
	}
	now := time.Now()
	n := 0
	for _, t := range f.Tasks {
		if t.Status == store.StatusDone && !all {
			continue
		}
		mark := "○"
		if t.Status == store.StatusDone {
			mark = "✓"
		}
		fmt.Printf("%s %3d  %s\n", mark, t.ID, describe(t, now))
		n++
	}
	if n == 0 {
		fmt.Println("남은 일 없음")
	}
	return nil
}

func cmdDone(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("완료할 id가 없다. 예: tasklet done 3")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("id는 숫자여야 한다: %q", args[0])
	}
	var title string
	if err := store.Update(func(f *store.File) error {
		t := f.Find(id)
		if t == nil {
			return fmt.Errorf("%d번 할 일이 없다", id)
		}
		title = t.Title
		return f.Complete(id, time.Now())
	}); err != nil {
		return err
	}
	if jsonMode {
		f, err := store.Read()
		if err != nil {
			return err
		}
		return printState(f, nil, id)
	}
	fmt.Printf("%d번 완료: %s\n", id, title)
	return nil
}

// cmdDelete는 할 일을 목록에서 지운다. 확인은 부르는 쪽(팝오버 🗑)에서 받는다.
func cmdDelete(args []string) error {
	id, err := idArg(args, "지울")
	if err != nil {
		return err
	}
	var title string
	if err := store.Update(func(f *store.File) error {
		t := f.Find(id)
		if t == nil {
			return fmt.Errorf("%d번 할 일이 없다", id)
		}
		title = t.Title
		return f.Delete(id)
	}); err != nil {
		return err
	}
	if jsonMode {
		f, err := store.Read()
		if err != nil {
			return err
		}
		// 지운 id는 더 이상 없으므로 changed로 넘기지 않는다.
		return printState(f, nil, 0)
	}
	fmt.Printf("%d번 삭제: %s\n", id, title)
	return nil
}

// describe는 한 줄 요약: "회원가입 기능 개발  ·  성민우 선임  ·  9/29(화) D-6"
func describe(t store.Task, now time.Time) string {
	parts := []string{t.Title}
	if t.Requester != nil && *t.Requester != "" {
		parts = append(parts, *t.Requester)
	}
	parts = append(parts, dueLabel(t.DueAt, now))
	return strings.Join(parts, "  ·  ")
}

func dueLabel(dueAt *string, now time.Time) string {
	if dueAt == nil {
		return "기한 없음"
	}
	d, err := time.ParseInLocation(time.DateOnly, *dueAt, now.Location())
	if err != nil {
		return *dueAt
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(d.Sub(today).Hours() / 24)
	label := fmt.Sprintf("%d/%d(%s)", int(d.Month()), d.Day(), [...]string{"일", "월", "화", "수", "목", "금", "토"}[d.Weekday()])
	switch {
	case days < 0:
		return fmt.Sprintf("%s %d일 지남", label, -days)
	case days == 0:
		return label + " 오늘"
	case days == 1:
		return label + " 내일"
	default:
		return fmt.Sprintf("%s D-%d", label, days)
	}
}

func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func datePtr(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

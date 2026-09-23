// tasklet — 메뉴바 업무 메모 (DESIGN.md 4장)
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SungMinWoo/tasklet/internal/parse"
	"github.com/SungMinWoo/tasklet/internal/store"
)

const usage = `사용법:
  tasklet add "<자연어>"   할 일 추가 (Haiku 파싱, 3~12초)
  tasklet list             남은 일 목록
  tasklet list --all       완료한 일까지
  tasklet done <id>        완료 처리
`

func main() {
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
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "모르는 명령: %s\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "오류:", err)
		os.Exit(1)
	}
}

func cmdAdd(args []string) error {
	sentence := strings.TrimSpace(strings.Join(args, " "))
	if sentence == "" {
		return fmt.Errorf(`추가할 문장이 없다. 예: tasklet add "화요일까지 회원가입 개발"`)
	}
	if err := parse.LookupClaude(); err != nil {
		return err
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

	task := store.Task{
		Title:     res.Title,
		Requester: strPtr(res.From),
		DueAt:     datePtr(resolved.Date),
		Raw:       sentence,
		CreatedAt: today,
	}
	var id int
	if err := store.Update(func(f *store.File) error {
		id = f.Add(task)
		return nil
	}); err != nil {
		return err
	}

	fmt.Printf("%d번 추가: %s\n", id, describe(task, today))
	// 확인창은 메뉴에서 띄운다. 터미널에서는 이유만 알린다 (DESIGN.md 8장).
	for _, r := range parse.Reasons(sentence, res, resolved) {
		fmt.Printf("  · %s\n", r)
	}
	return nil
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
	fmt.Printf("%d번 완료: %s\n", id, title)
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

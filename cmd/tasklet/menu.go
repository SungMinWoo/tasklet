package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SungMinWoo/tasklet/internal/parse"
	"github.com/SungMinWoo/tasklet/internal/render"
	"github.com/SungMinWoo/tasklet/internal/store"
)

// cmdMenu는 SwiftBar가 10초마다 부른다. 절대 빈손으로 끝내지 않는다.
func cmdMenu() error {
	f, err := store.Read()
	if err != nil {
		// 파일이 깨져도 메뉴는 떠야 한다.
		fmt.Printf("읽기 실패 | color=red\n---\n%s\n", err)
		return nil
	}
	fmt.Print(render.Menu(f, store.LoadConfig(), time.Now(), exePath()))
	return nil
}

// cmdPrompt는 '+ 추가하기'. 입력창 → 파싱 → 애매하면 확인창 → 저장.
func cmdPrompt() error {
	sentence, err := ask("할 일을 한 줄로 적어주세요", "")
	if err != nil {
		return err
	}
	if sentence == "" {
		return nil
	}
	return addInteractive(sentence, 0)
}

// cmdEdit은 '수정'. 원래 문장이 채워진 입력창에서 시작한다.
func cmdEdit(args []string) error {
	id, err := idArg(args, "수정할")
	if err != nil {
		return err
	}
	f, err := store.Read()
	if err != nil {
		return err
	}
	t := f.Find(id)
	if t == nil {
		return fmt.Errorf("%d번 할 일이 없다", id)
	}
	sentence, err := ask("고쳐 쓰기", t.Raw)
	if err != nil {
		return err
	}
	if sentence == "" || sentence == t.Raw {
		return nil
	}
	return addInteractive(sentence, id)
}

// addInteractive는 입력 문장을 해석해 저장한다.
// 애매하면 확인창을 띄우고, '아니오'면 한 번 더 고쳐 쓰게 한 뒤 그때는 묻지 않는다
// (무한 반복 방지, DESIGN.md 8장).
// id가 0이면 새로 추가, 아니면 그 할 일을 덮어쓴다.
func addInteractive(sentence string, id int) error {
	for attempt := 0; ; attempt++ {
		res, resolved, err := interpret(sentence)
		if err != nil {
			return err
		}
		reasons := parse.Reasons(sentence, res, resolved)

		if len(reasons) > 0 && attempt == 0 {
			ok, err := confirm(summary(res, resolved) + "\n" + strings.Join(reasons, "\n"))
			if err != nil {
				return err
			}
			if !ok {
				fixed, err := ask("고쳐 쓰기", sentence)
				if err != nil {
					return err
				}
				if fixed == "" {
					return nil
				}
				sentence = fixed
				continue
			}
		}
		return save(sentence, res, resolved, id)
	}
}

func save(sentence string, res parse.Result, resolved parse.Resolved, id int) error {
	if id == 0 {
		newID, err := saveNew(sentence, res, resolved)
		if err != nil {
			return err
		}
		notify(fmt.Sprintf("%d번 추가: %s", newID, summary(res, resolved)))
		return nil
	}
	// 수정: id·상태·만든 시각은 유지하고 내용만 갈아끼운다 (DESIGN.md 8장).
	err := store.Update(func(f *store.File) error {
		t := f.Find(id)
		if t == nil {
			return fmt.Errorf("%d번 할 일이 없다", id)
		}
		t.Title = res.Title
		t.Requester = strPtr(res.From)
		t.DueAt = datePtr(resolved.Date)
		t.Raw = sentence
		return nil
	})
	if err != nil {
		return err
	}
	notify(fmt.Sprintf("%d번 수정: %s", id, summary(res, resolved)))
	return nil
}

// summary는 확인창·알림에 쓸 한 줄. 날짜는 Go가 계산한 실제 날짜로 보여준다.
func summary(res parse.Result, resolved parse.Resolved) string {
	parts := []string{res.Title}
	if strings.TrimSpace(res.From) != "" {
		parts = append(parts, res.From)
	}
	parts = append(parts, dueLabel(datePtr(resolved.Date), time.Now()))
	return strings.Join(parts, " · ")
}

// cmdDue는 '기한 ▸'. Claude를 부르지 않고 Go가 바로 계산한다.
func cmdDue(args []string) error {
	id, err := idArg(args, "기한을 바꿀")
	if err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("기한 값이 없다. 예: tasklet due 3 +1d")
	}
	due, err := parseSpec(args[1])
	if err != nil {
		return err
	}
	resolved, err := parse.ResolveDue(due, time.Now())
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
		t.DueAt = datePtr(resolved.Date)
		return nil
	}); err != nil {
		return err
	}
	fmt.Printf("%d번 기한: %s · %s\n", id, title, dueLabel(datePtr(resolved.Date), time.Now()))
	return nil
}

// parseSpec은 메뉴가 넘기는 기한 값을 Due로 바꾼다 (6장 kind 표기 그대로).
func parseSpec(s string) (parse.Due, error) {
	switch s {
	case "none", "eow", "next_eow", "eom":
		return parse.Due{Kind: s}, nil
	}
	if strings.HasPrefix(s, "+") {
		return parse.Due{Kind: "relative", Value: s}, nil
	}
	if len(s) == 3 { // mon, tue, ...
		return parse.Due{Kind: "weekday", Value: s}, nil
	}
	return parse.Due{Kind: "date", Value: s}, nil
}

// cmdSetting은 theme·mascot 변경. 모르는 값이면 고를 수 있는 목록을 보여준다.
func cmdSetting(kind string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s 값이 없다. 고를 수 있는 값: %s", kind, strings.Join(choices(kind), " "))
	}
	name := args[0]
	if !contains(choices(kind), name) {
		return fmt.Errorf("모르는 %s: %q. 고를 수 있는 값: %s", kind, name, strings.Join(choices(kind), " "))
	}
	c := store.LoadConfig()
	if kind == "theme" {
		c.Theme = name
	} else {
		c.Mascot = name
	}
	if err := store.SaveConfig(c); err != nil {
		return err
	}
	fmt.Printf("%s: %s\n", kind, name)
	return nil
}

func choices(kind string) []string {
	if kind == "theme" {
		return store.Themes
	}
	return store.Mascots
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func idArg(args []string, what string) (int, error) {
	if len(args) < 1 {
		return 0, fmt.Errorf("%s id가 없다", what)
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, fmt.Errorf("id는 숫자여야 한다: %q", args[0])
	}
	return id, nil
}

// exePath는 메뉴의 클릭 동작에 넣을 실행 파일 경로.
func exePath() string {
	p, err := os.Executable()
	if err != nil {
		return "tasklet"
	}
	return p
}

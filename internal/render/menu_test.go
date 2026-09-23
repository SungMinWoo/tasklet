package render

import (
	"strings"
	"testing"

	"github.com/SungMinWoo/tasklet/internal/store"
)

func withTitle(id int, title, due, requester string) store.Task {
	t := task(id, due)
	t.Title = title
	if requester != "" {
		r := requester
		t.Requester = &r
	}
	return t
}

func cfg() store.Config {
	return store.Config{Theme: "mono", Mascot: "pig", WeekendDueBucket: "later"}
}

func TestMenuTitle(t *testing.T) {
	now := day("2026-09-23")
	tests := []struct {
		name  string
		tasks []store.Task
		want  string
	}{
		{"지남 있음", []store.Task{task(1, "2026-09-20"), task(2, "2026-09-23")}, "1 · 오늘 1"},
		{"지남 없음", []store.Task{task(1, "2026-09-23"), task(2, "2026-09-23")}, "오늘 2"},
		{"남은 일 없음", nil, "클리어"},
		{"오늘·지남은 없지만 남은 일이 있으면 개수", []store.Task{task(1, ""), task(2, "2026-09-25")}, "남음 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := strings.SplitN(Menu(&store.File{Tasks: tt.tasks}, cfg(), now, "/bin/tasklet"), "\n", 2)[0]
			text := strings.SplitN(line, " | ", 2)[0]
			if text != tt.want {
				t.Errorf("타이틀 = %q, want %q", text, tt.want)
			}
			if !strings.Contains(line, "image=") {
				t.Errorf("타이틀에 캐릭터 그림이 없다: %q", line)
			}
		})
	}
}

func TestMenuStructure(t *testing.T) {
	now := day("2026-09-23") // 수요일
	f := &store.File{Tasks: []store.Task{
		withTitle(9, "로그인 버그 수정", "2026-09-21", ""),
		withTitle(12, "회원가입 기능 개발", "2026-09-23", "성민우 선임"),
		withTitle(3, "로그 대시보드 개선", "", ""),
	}}
	out := Menu(f, cfg(), now, "/x/tasklet")
	lines := strings.Split(out, "\n")

	// 버킷 5개가 순서대로 있어야 한다
	var buckets []string
	for _, l := range lines {
		for _, k := range BucketOrder {
			if strings.HasPrefix(l, k.Label()) {
				buckets = append(buckets, k.Label())
			}
		}
	}
	want := []string{"지남", "오늘", "내일", "이번주", "나중"}
	if strings.Join(buckets, ",") != strings.Join(want, ",") {
		t.Errorf("버킷 줄 = %v, want %v", buckets, want)
	}

	for _, want := range []string{
		"--로그인 버그 수정 · -2일\n",
		"--회원가입 기능 개발 · 성민우 선임 · 오늘\n",
		"--로그 대시보드 개선\n",
		`----완료 | bash="/x/tasklet" param1=done param2=12`,
		`----수정 | bash="/x/tasklet" param1=edit param2=12`,
		"----기한\n",
		`------내일 | bash="/x/tasklet" param1=due param2=3 param3=+1d`,
		`------기한 없음 | bash="/x/tasklet" param1=due param2=3 param3=none`,
		`+ 추가하기 | bash="/x/tasklet" param1=prompt`,
		"--✓ 원 포인트 | ",
		"--   타이드 | ",
		"--✓ 돼지 | ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("메뉴에 없음: %q\n---\n%s", want, out)
		}
	}
}

func TestMenuEscapesPipe(t *testing.T) {
	f := &store.File{Tasks: []store.Task{withTitle(1, "A | B 정리", "", "")}}
	out := Menu(f, cfg(), day("2026-09-23"), "/x/tasklet")
	if strings.Contains(out, "--A | B 정리") {
		t.Errorf("제목의 | 가 그대로 나가면 SwiftBar가 파라미터로 읽는다:\n%s", out)
	}
}

func TestBarLength(t *testing.T) {
	if got := bar(3); got != "▇▇▇" {
		t.Errorf("bar(3) = %q", got)
	}
	if got := bar(20); !strings.HasSuffix(got, "…") || len([]rune(got)) != 13 {
		t.Errorf("bar(20) = %q, 12개에서 잘라야 한다", got)
	}
}

package render

import (
	"testing"
	"time"

	"github.com/SungMinWoo/tasklet/internal/store"
)

func doneAt(id int, at string) store.Task {
	ts, err := time.ParseInLocation("2006-01-02 15:04", at, time.UTC)
	if err != nil {
		panic(err)
	}
	return store.Task{ID: id, Title: "일", Status: store.StatusDone, DoneAt: &ts}
}

func TestDoneByDay(t *testing.T) {
	noTime := store.Task{ID: 9, Status: store.StatusDone} // 완료 시각 없음 → 뺀다
	tasks := []store.Task{
		doneAt(1, "2026-10-17 09:00"),
		task(2, "2026-10-20"), // 아직 안 한 일 → 뺀다
		doneAt(3, "2026-10-19 08:00"),
		doneAt(4, "2026-10-17 18:30"),
		noTime,
		doneAt(5, "2026-10-17 23:59"),
		doneAt(6, "2026-10-18 00:00"), // 자정은 다음 날
	}
	days := DoneByDay(tasks, time.UTC)

	want := []struct {
		date string
		ids  []int
	}{
		{"2026-10-19", []int{3}},
		{"2026-10-18", []int{6}},
		{"2026-10-17", []int{5, 4, 1}},
	}
	if len(days) != len(want) {
		t.Fatalf("날짜 %d개, want %d개: %+v", len(days), len(want), days)
	}
	for i, w := range want {
		if got := days[i].Date.Format(time.DateOnly); got != w.date {
			t.Errorf("days[%d] = %s, want %s", i, got, w.date)
		}
		var ids []int
		for _, task := range days[i].Tasks {
			ids = append(ids, task.ID)
		}
		if len(ids) != len(w.ids) {
			t.Errorf("%s ids = %v, want %v", w.date, ids, w.ids)
			continue
		}
		for j := range ids {
			if ids[j] != w.ids[j] {
				t.Errorf("%s ids = %v, want %v", w.date, ids, w.ids)
				break
			}
		}
	}
}

func TestDoneByDayLocalDate(t *testing.T) {
	// UTC로는 10/17 15:30이지만 서울(+9)로는 10/18 00:30 → 10/18로 묶여야 한다.
	seoul := time.FixedZone("KST", 9*3600)
	days := DoneByDay([]store.Task{doneAt(1, "2026-10-17 15:30")}, seoul)
	if len(days) != 1 || days[0].Date.Format(time.DateOnly) != "2026-10-18" {
		t.Fatalf("got %+v, want 2026-10-18", days)
	}
}

func TestDoneByDayEmpty(t *testing.T) {
	if days := DoneByDay([]store.Task{task(1, "")}, time.UTC); len(days) != 0 {
		t.Fatalf("got %+v, want 없음", days)
	}
}

func TestDayLabel(t *testing.T) {
	now := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC) // 월
	tests := []struct {
		day, want string
	}{
		{"2026-10-19", "10/19(월) 오늘"},
		{"2026-10-18", "10/18(일) 어제"},
		{"2026-10-17", "10/17(토)"},
		{"2025-12-31", "2025/12/31(수)"},
	}
	for _, tt := range tests {
		if got := DayLabel(day(tt.day), now); got != tt.want {
			t.Errorf("DayLabel(%s) = %q, want %q", tt.day, got, tt.want)
		}
	}
}

func TestCreatedByDay(t *testing.T) {
	created := func(id int, at string, done bool) store.Task {
		ts, err := time.ParseInLocation("2006-01-02 15:04", at, time.UTC)
		if err != nil {
			panic(err)
		}
		s := store.StatusTodo
		if done {
			s = store.StatusDone
		}
		return store.Task{ID: id, Status: s, CreatedAt: ts}
	}
	tasks := []store.Task{
		created(1, "2026-10-17 09:00", true), // 완료한 것도 들어간다
		created(2, "2026-10-19 10:00", false),
		created(3, "2026-10-17 18:00", false),
		{ID: 4, Status: store.StatusTodo}, // 등록 시각 없음 → 뺀다
	}
	days := CreatedByDay(tasks, time.UTC)
	if len(days) != 2 {
		t.Fatalf("날짜 %d개, want 2: %+v", len(days), days)
	}
	if got := days[0].Date.Format(time.DateOnly); got != "2026-10-19" || len(days[0].Tasks) != 1 || days[0].Tasks[0].ID != 2 {
		t.Errorf("days[0] = %s %+v", got, days[0].Tasks)
	}
	if got := days[1].Date.Format(time.DateOnly); got != "2026-10-17" || len(days[1].Tasks) != 2 ||
		days[1].Tasks[0].ID != 3 || days[1].Tasks[1].ID != 1 {
		t.Errorf("days[1] = %s %+v", got, days[1].Tasks)
	}
}

func TestBetween(t *testing.T) {
	days := []Day{{Date: day("2026-10-19")}, {Date: day("2026-10-01")}, {Date: day("2026-09-19")}, {Date: day("2026-09-18")}}
	got := Between(days, day("2026-09-19"), time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC))
	if len(got) != 3 || !got[0].Date.Equal(day("2026-10-19")) || !got[2].Date.Equal(day("2026-09-19")) {
		t.Fatalf("got %+v, want 10/19 · 10/1 · 9/19 (양 끝 포함)", got)
	}
}

func TestMonthBefore(t *testing.T) {
	tests := []struct{ in, want string }{
		{"2026-10-19", "2026-09-19"},
		{"2026-03-31", "2026-02-28"},
		{"2028-03-31", "2028-02-29"}, // 윤년
		{"2026-01-15", "2025-12-15"},
	}
	for _, tt := range tests {
		if got := MonthBefore(day(tt.in)).Format(time.DateOnly); got != tt.want {
			t.Errorf("MonthBefore(%s) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

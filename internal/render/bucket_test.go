package render

import (
	"testing"
	"time"

	"github.com/SungMinWoo/tasklet/internal/store"
)

func task(id int, due string) store.Task {
	t := store.Task{ID: id, Title: "일", Status: store.StatusTodo}
	if due != "" {
		d := due
		t.DueAt = &d
	}
	return t
}

func day(s string) time.Time {
	d, err := time.ParseInLocation(time.DateOnly, s, time.UTC)
	if err != nil {
		panic(err)
	}
	return d
}

func TestBucketOf(t *testing.T) {
	// 2026-09-21 월, 23 수, 25 금, 26 토, 27 일, 28 월, 10-02 금
	tests := []struct {
		name    string
		today   string
		due     string
		weekend string
		want    BucketKey
	}{
		{"기한 없음", "2026-09-23", "", "later", Later},
		{"어제", "2026-09-23", "2026-09-22", "later", Past},
		{"한참 전", "2026-09-23", "2026-08-01", "later", Past},
		{"오늘", "2026-09-23", "2026-09-23", "later", Today},
		{"내일", "2026-09-23", "2026-09-24", "later", Tomorrow},
		{"모레 = 이번주", "2026-09-23", "2026-09-25", "later", ThisWeek},
		{"이번 주 금요일", "2026-09-23", "2026-09-25", "later", ThisWeek},
		{"이번 주 토요일 → 기본은 나중", "2026-09-23", "2026-09-26", "later", Later},
		{"이번 주 토요일 → this_week 설정이면 이번주", "2026-09-23", "2026-09-26", "this_week", ThisWeek},
		{"다음 주 월요일 → 나중", "2026-09-23", "2026-09-28", "later", Later},

		{"월요일에 이번 주 금요일", "2026-09-21", "2026-09-25", "later", ThisWeek},
		{"금요일에 다음 주 월요일 → 나중", "2026-09-25", "2026-09-28", "later", Later},
		{"금요일에 내일(토)", "2026-09-25", "2026-09-26", "later", Tomorrow},

		// 주말에 볼 때 '이번주' = 다음 주 월~금 (DESIGN.md 7장)
		{"토요일에 다음 주 월요일 → 이번주", "2026-09-26", "2026-09-28", "later", ThisWeek},
		{"토요일에 내일(일)", "2026-09-26", "2026-09-27", "later", Tomorrow},
		{"토요일에 다음 주 금요일 → 이번주", "2026-09-26", "2026-10-02", "later", ThisWeek},
		{"토요일에 다다음 주 월요일 → 나중", "2026-09-26", "2026-10-05", "later", Later},
		{"일요일에 내일(월)", "2026-09-27", "2026-09-28", "later", Tomorrow},
		{"일요일에 다음 주 화요일 → 이번주", "2026-09-27", "2026-09-29", "later", ThisWeek},
		{"일요일에 다음 주 금요일 → 이번주", "2026-09-27", "2026-10-02", "later", ThisWeek},

		{"깨진 날짜 값은 나중", "2026-09-23", "언젠가", "later", Later},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BucketOf(task(1, tt.due), day(tt.today), tt.weekend)
			if got != tt.want {
				t.Errorf("버킷 = %s, want %s", got.Label(), tt.want.Label())
			}
		})
	}
}

func TestBucketsSortsAndSkipsDone(t *testing.T) {
	done := task(9, "2026-09-23")
	done.Status = store.StatusDone
	tasks := []store.Task{
		task(1, "2026-09-25"),
		task(2, ""),
		task(3, "2026-09-24"),
		done,
		task(4, ""),
	}
	got := Buckets(tasks, day("2026-09-23"), "later")

	if len(got[Today]) != 0 {
		t.Errorf("완료한 일이 오늘 버킷에 들어갔다: %+v", got[Today])
	}
	if len(got[Tomorrow]) != 1 || got[Tomorrow][0].ID != 3 {
		t.Errorf("내일 버킷 = %+v", got[Tomorrow])
	}
	if len(got[Later]) != 2 || got[Later][0].ID != 2 || got[Later][1].ID != 4 {
		t.Errorf("기한 없는 일은 id 순이어야 한다: %+v", got[Later])
	}
}

func TestBucketsOrderByDueThenID(t *testing.T) {
	tasks := []store.Task{
		task(5, "2026-09-25"),
		task(2, "2026-09-25"),
		task(7, "2026-09-24"),
	}
	got := Buckets(tasks, day("2026-09-20"), "later") // 일요일 → 전부 이번주
	ids := []int{}
	for _, t := range got[ThisWeek] {
		ids = append(ids, t.ID)
	}
	want := []int{7, 2, 5}
	if len(ids) != len(want) {
		t.Fatalf("이번주 = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("정렬 = %v, want %v (기한 순 → id 순)", ids, want)
		}
	}
}

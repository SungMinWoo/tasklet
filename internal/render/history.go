package render

import (
	"fmt"
	"sort"
	"time"

	"github.com/SungMinWoo/tasklet/internal/store"
)

// Day는 기록의 하루치. Date는 그날 0시 (loc 시간대).
type Day struct {
	Date  time.Time
	Tasks []store.Task
}

// DoneByDay는 완료한 일을 완료한 날짜별로 묶는다.
// 완료 시각이 없는 것(사람이 파일을 고친 경우)은 날짜를 알 수 없어 뺀다.
func DoneByDay(tasks []store.Task, loc *time.Location) []Day {
	return byDay(tasks, loc, func(t store.Task) *time.Time {
		if t.Status != store.StatusDone {
			return nil
		}
		return t.DoneAt
	})
}

// CreatedByDay는 완료 여부와 상관없이 모든 일을 등록한 날짜별로 묶는다.
func CreatedByDay(tasks []store.Task, loc *time.Location) []Day {
	return byDay(tasks, loc, func(t store.Task) *time.Time {
		if t.CreatedAt.IsZero() {
			return nil
		}
		return &t.CreatedAt
	})
}

// byDay는 at이 주는 시각으로 날짜별로 묶는다. at이 nil이면 뺀다.
// 최근 날짜가 위, 하루 안에서도 최근 것이 위.
func byDay(tasks []store.Task, loc *time.Location, at func(store.Task) *time.Time) []Day {
	type stamped struct {
		task store.Task
		at   time.Time
	}
	var list []stamped
	for _, t := range tasks {
		if ts := at(t); ts != nil {
			list = append(list, stamped{t, *ts})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].at.After(list[j].at) })

	var out []Day
	for _, s := range list {
		d := midnight(s.at.In(loc))
		if n := len(out); n > 0 && out[n-1].Date.Equal(d) {
			out[n-1].Tasks = append(out[n-1].Tasks, s.task)
			continue
		}
		out = append(out, Day{Date: d, Tasks: []store.Task{s.task}})
	}
	return out
}

// Between은 from~to(양 끝 포함, 날짜 단위) 안의 날만 남긴다.
func Between(days []Day, from, to time.Time) []Day {
	from, to = midnight(from), midnight(to)
	var out []Day
	for _, d := range days {
		if !d.Date.Before(from) && !d.Date.After(to) {
			out = append(out, d)
		}
	}
	return out
}

// MonthBefore는 한 달 전 같은 날. 그런 날이 없으면 그 달 마지막 날 (3/31 → 2/28).
// time.AddDate는 2/31을 3/3으로 넘겨 버린다.
func MonthBefore(t time.Time) time.Time {
	first := time.Date(t.Year(), t.Month()-1, 1, 0, 0, 0, 0, t.Location())
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(t.Day(), last), 0, 0, 0, 0, t.Location())
}

// DayLabel은 "10/17(금)". 오늘·어제는 뒤에 붙이고, 올해가 아니면 연도를 앞에 붙인다.
func DayLabel(d, now time.Time) string {
	label := fmt.Sprintf("%d/%d(%s)", int(d.Month()), d.Day(), [...]string{"일", "월", "화", "수", "목", "금", "토"}[d.Weekday()])
	if d.Year() != now.Year() {
		label = fmt.Sprintf("%d/%s", d.Year(), label)
	}
	today := midnight(now)
	switch {
	case d.Equal(today):
		return label + " 오늘"
	case d.Equal(today.AddDate(0, 0, -1)):
		return label + " 어제"
	}
	return label
}

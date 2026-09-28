// Package parse는 Haiku가 뽑은 기한 표현을 실제 날짜로 바꾼다.
// 날짜 계산은 LLM에 맡기지 않는다.
package parse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Due는 Haiku 응답의 due 필드. Value는 kind에 따라 비어 있을 수 있다.
type Due struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Resolved는 계산된 기한.
type Resolved struct {
	Date   time.Time // IsZero()면 기한 없음
	Reason string    // 비어 있지 않으면 확인창에 보여줄 애매함 사유
}

var weekdays = map[string]time.Weekday{
	"mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday, "thu": time.Thursday,
	"fri": time.Friday, "sat": time.Saturday, "sun": time.Sunday,
}

var weekdayKo = [...]string{"일", "월", "화", "수", "목", "금", "토"}

var relativeRe = regexp.MustCompile(`^\+(\d+)d$`)

// ResolveDue는 d를 today 기준 실제 날짜로 바꾼다. today의 시각은 무시한다.
func ResolveDue(d Due, today time.Time) (Resolved, error) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	v := strings.TrimSpace(d.Value)

	switch d.Kind {
	case "none":
		return Resolved{}, nil

	case "relative":
		m := relativeRe.FindStringSubmatch(v)
		if m == nil {
			return Resolved{}, fmt.Errorf("relative 값 형식 오류: %q", d.Value)
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return Resolved{}, fmt.Errorf("relative 값 형식 오류: %q", d.Value)
		}
		return Resolved{Date: today.AddDate(0, 0, n)}, nil

	case "weekday":
		wd, ok := weekdays[v]
		if !ok {
			return Resolved{}, fmt.Errorf("weekday 값 오류: %q", d.Value)
		}
		diff := (int(wd) - int(today.Weekday()) + 7) % 7
		if diff == 0 {
			date := today.AddDate(0, 0, 7)
			return Resolved{Date: date, Reason: fmt.Sprintf("%s요일 = 오늘이라 다음 주 %s로 계산", weekdayKo[wd], short(date, today))}, nil
		}
		return Resolved{Date: today.AddDate(0, 0, diff)}, nil

	case "next_weekday":
		wd, ok := weekdays[v]
		if !ok {
			return Resolved{}, fmt.Errorf("next_weekday 값 오류: %q", d.Value)
		}
		return Resolved{Date: nextMonday(today).AddDate(0, 0, mondayIndex(wd))}, nil

	case "eow":
		// 평일: 이번 주 금요일. 주말: 다음 주 금요일.
		if idx := mondayIndex(today.Weekday()); idx <= 4 {
			return Resolved{Date: today.AddDate(0, 0, 4-idx)}, nil
		}
		return Resolved{Date: nextMonday(today).AddDate(0, 0, 4)}, nil

	case "next_eow":
		date := nextMonday(today).AddDate(0, 0, 4)
		if isWeekend(today) {
			// 주말엔 eow도 같은 날이 되므로 사용자에게 확인한다 (A5).
			return Resolved{Date: date, Reason: fmt.Sprintf("주말이라 '이번 주'와 '다음 주'가 같은 %s로 계산", short(date, today))}, nil
		}
		return Resolved{Date: date}, nil

	case "eom":
		return Resolved{Date: time.Date(today.Year(), today.Month()+1, 0, 0, 0, 0, 0, today.Location())}, nil

	case "date":
		return resolveDate(v, today)
	}
	return Resolved{}, fmt.Errorf("알 수 없는 kind: %q", d.Kind)
}

// resolveDate는 DD / MM-DD / YYYY-MM-DD를 처리한다.
// 빠진 월·연도는 채우고, 채운 날짜가 오늘 이전이면 다음 달 / 내년으로 넘긴다.
// 그 달에 없는 날짜는 말일로 당긴다.
func resolveDate(v string, today time.Time) (Resolved, error) {
	parts := strings.Split(v, "-")
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return Resolved{}, fmt.Errorf("date 값 형식 오류: %q", v)
		}
		nums[i] = n
	}
	loc := today.Location()

	switch len(nums) {
	case 1: // DD
		day := nums[0]
		if day < 1 || day > 31 {
			return Resolved{}, fmt.Errorf("date 일 범위 오류: %q", v)
		}
		date, clamped := clampDate(today.Year(), today.Month(), day, loc)
		rolled := false
		if date.Before(today) {
			date, clamped = clampDate(today.Year(), today.Month()+1, day, loc)
			rolled = true
		}
		return Resolved{Date: date, Reason: rollReason(day, date, today, rolled, clamped)}, nil

	case 2: // MM-DD
		month, day := nums[0], nums[1]
		if month < 1 || month > 12 || day < 1 || day > 31 {
			return Resolved{}, fmt.Errorf("date 월·일 범위 오류: %q", v)
		}
		date, clamped := clampDate(today.Year(), time.Month(month), day, loc)
		rolled := false
		if date.Before(today) {
			date, clamped = clampDate(today.Year()+1, time.Month(month), day, loc)
			rolled = true
		}
		return Resolved{Date: date, Reason: rollReason(day, date, today, rolled, clamped)}, nil

	case 3: // YYYY-MM-DD: 그대로. 지났으면 '지남' 버킷으로 간다.
		date := time.Date(nums[0], time.Month(nums[1]), nums[2], 0, 0, 0, 0, loc)
		if date.Year() != nums[0] || int(date.Month()) != nums[1] || date.Day() != nums[2] {
			return Resolved{}, fmt.Errorf("존재하지 않는 날짜: %q", v)
		}
		return Resolved{Date: date}, nil
	}
	return Resolved{}, fmt.Errorf("date 값 형식 오류: %q", v)
}

// clampDate는 year-month의 day를 만들되, 그 달에 없는 날이면 말일로 당긴다.
// month가 13이어도 time.Date 규칙대로 다음 해 1월로 넘어간다.
func clampDate(year int, month time.Month, day int, loc *time.Location) (time.Time, bool) {
	first := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	last := first.AddDate(0, 1, -1).Day()
	if day > last {
		return first.AddDate(0, 0, last-1), true
	}
	return first.AddDate(0, 0, day-1), false
}

func rollReason(day int, date, today time.Time, rolled, clamped bool) string {
	switch {
	case rolled && clamped:
		return fmt.Sprintf("이미 지난 날짜이고 %d일이 없는 달이라 %s로 계산", day, short(date, today))
	case rolled:
		return fmt.Sprintf("이미 지난 날짜라 %s로 계산", short(date, today))
	case clamped:
		return fmt.Sprintf("%d일이 없는 달이라 말일 %s로 계산", day, short(date, today))
	}
	return ""
}

// mondayIndex: 월=0 … 일=6.
func mondayIndex(wd time.Weekday) int { return (int(wd) + 6) % 7 }

// nextMonday는 월요일에 시작하는 바로 다음 주의 월요일.
func nextMonday(today time.Time) time.Time {
	return today.AddDate(0, 0, 7-mondayIndex(today.Weekday()))
}

func isWeekend(t time.Time) bool { return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday }

// short는 확인창용 날짜 표기: 9/29(화). 올해가 아니면 2027/10/2(토).
func short(t, today time.Time) string {
	s := fmt.Sprintf("%d/%d(%s)", int(t.Month()), t.Day(), weekdayKo[t.Weekday()])
	if t.Year() != today.Year() {
		return fmt.Sprintf("%d/%s", t.Year(), s)
	}
	return s
}

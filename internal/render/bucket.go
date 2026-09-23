// Package render는 할 일을 버킷으로 나누고 SwiftBar 메뉴 텍스트를 만든다 (DESIGN.md 7장).
package render

import (
	"sort"
	"time"

	"github.com/SungMinWoo/tasklet/internal/store"
)

type BucketKey int

const (
	Past BucketKey = iota // 지남
	Today
	Tomorrow
	ThisWeek
	Later
)

var BucketOrder = []BucketKey{Past, Today, Tomorrow, ThisWeek, Later}

var bucketLabel = map[BucketKey]string{
	Past: "지남", Today: "오늘", Tomorrow: "내일", ThisWeek: "이번주", Later: "나중",
}

func (b BucketKey) Label() string { return bucketLabel[b] }

// Buckets는 완료되지 않은 일만 버킷별로 나눈다. 각 버킷 안은 기한 빠른 순,
// 기한이 같으면 id 순 (메뉴 순서가 호출마다 흔들리지 않게).
func Buckets(tasks []store.Task, now time.Time, weekendBucket string) map[BucketKey][]store.Task {
	out := map[BucketKey][]store.Task{}
	for _, t := range tasks {
		if t.Status == store.StatusDone {
			continue
		}
		out[BucketOf(t, now, weekendBucket)] = append(out[BucketOf(t, now, weekendBucket)], t)
	}
	for k := range out {
		list := out[k]
		sort.SliceStable(list, func(i, j int) bool {
			di, dj := dueOf(list[i]), dueOf(list[j])
			switch {
			case di == "" && dj != "":
				return false
			case di != "" && dj == "":
				return true
			case di != dj:
				return di < dj
			}
			return list[i].ID < list[j].ID
		})
		out[k] = list
	}
	return out
}

func dueOf(t store.Task) string {
	if t.DueAt == nil {
		return ""
	}
	return *t.DueAt
}

// BucketOf는 할 일 하나가 어느 버킷인지 정한다 (DESIGN.md 7장).
//
//	지남   기한 < 오늘
//	오늘   기한 = 오늘
//	내일   기한 = 내일
//	이번주 모레 ~ 이번 주 금요일. 주말에 볼 때는 다음 주 월~금
//	나중   그 밖 + 기한 없음. 기한이 토·일이면 weekendBucket 설정에 따름
func BucketOf(t store.Task, now time.Time, weekendBucket string) BucketKey {
	if t.DueAt == nil {
		return Later
	}
	due, err := time.ParseInLocation(time.DateOnly, *t.DueAt, now.Location())
	if err != nil {
		return Later // 사람이 파일을 잘못 고친 경우
	}
	today := midnight(now)
	switch {
	case due.Before(today):
		return Past
	case due.Equal(today):
		return Today
	case due.Equal(today.AddDate(0, 0, 1)):
		return Tomorrow
	}

	start, end := thisWeekWindow(today)
	if due.Before(start) || due.After(end) {
		return Later
	}
	if isWeekend(due) && weekendBucket != "this_week" {
		return Later
	}
	return ThisWeek
}

// thisWeekWindow는 '이번주' 버킷의 범위 (양 끝 포함).
// 평일: 모레 ~ 이번 주 금요일. 주말: 다음 주 월요일 ~ 금요일.
// weekendBucket이 this_week면 끝을 일요일까지 늘린다.
func thisWeekWindow(today time.Time) (start, end time.Time) {
	idx := mondayIndex(today.Weekday())
	if idx <= 4 { // 월~금
		return today.AddDate(0, 0, 2), today.AddDate(0, 0, 6-idx) // 6-idx = 이번 주 일요일
	}
	mon := today.AddDate(0, 0, 7-idx) // 다음 주 월요일
	return mon, mon.AddDate(0, 0, 6)  // 다음 주 일요일
}

func mondayIndex(w time.Weekday) int { return (int(w) + 6) % 7 }

func isWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
